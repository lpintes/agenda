// Package auth signs in to Google with the OAuth 2.0 flow for installed
// applications: the system browser, a loopback redirect and PKCE.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	clientFile = "client.json"
	tokenFile  = "token.dat"
	// plainTokenFile is the unencrypted token written by older versions.
	plainTokenFile = "token.json"
)

// Dir returns the directory holding the client file and the token,
// %APPDATA%\agenda on Windows.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "agenda"), nil
}

// LoadConfig reads the OAuth client from client.json in Dir.
func LoadConfig(scopes ...string) (*oauth2.Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, clientFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading OAuth client: %w", err)
	}
	cfg, err := google.ConfigFromJSON(data, scopes...)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// TokenSource returns a token source for cfg. It uses the stored token if
// there is one and signs in through the browser otherwise. Refreshed tokens
// are stored again.
func TokenSource(ctx context.Context, cfg *oauth2.Config) (oauth2.TokenSource, error) {
	tok, err := loadToken()
	if err != nil {
		tok, err = Login(ctx, cfg)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tok); err != nil {
			return nil, err
		}
	}
	return &savingSource{src: cfg.TokenSource(ctx, tok), last: tok}, nil
}

// Login opens the Google sign-in page in the default browser and waits for
// the redirect to a loopback address.
func Login(ctx context.Context, cfg *oauth2.Config) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer ln.Close()

	c := *cfg
	c.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/", ln.Addr().(*net.TCPAddr).Port)

	state, err := randomString()
	if err != nil {
		return nil, err
	}
	verifier := oauth2.GenerateVerifier()
	url := c.AuthCodeURL(state, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.S256ChallengeOption(verifier))

	type result struct {
		code string
		err  error
	}
	results := make(chan result, 1)
	var once sync.Once
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		var res result
		switch {
		case q.Get("state") != state:
			res.err = errors.New("sign-in failed: state mismatch")
		case q.Get("error") != "":
			res.err = fmt.Errorf("sign-in failed: %s", q.Get("error"))
		case q.Get("code") == "":
			res.err = errors.New("sign-in failed: no authorization code")
		default:
			res.code = q.Get("code")
		}
		msg := "Signed in to Agenda. You can close this window."
		if res.err != nil {
			msg = res.err.Error()
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"en\"><head><title>Agenda</title></head><body><h1>%s</h1></body></html>", html.EscapeString(msg))
		once.Do(func() { results <- res })
	})}
	go srv.Serve(ln)
	defer srv.Close()

	if err := openBrowser(url); err != nil {
		return nil, fmt.Errorf("opening browser: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-results:
		if res.err != nil {
			return nil, res.err
		}
		return c.Exchange(ctx, res.code, oauth2.VerifierOption(verifier))
	}
}

// savingSource stores the token whenever the underlying source refreshes it.
type savingSource struct {
	mu   sync.Mutex
	src  oauth2.TokenSource
	last *oauth2.Token
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, err := s.src.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last.AccessToken {
		if err := saveToken(tok); err != nil {
			return nil, err
		}
		s.last = tok
	}
	return tok, nil
}

// The token is stored as JSON encrypted with DPAPI, so only the current
// user on this computer can read it.

func tokenPath(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func loadToken() (*oauth2.Token, error) {
	path, err := tokenPath(tokenFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return migratePlainToken()
	}
	if err != nil {
		return nil, err
	}
	data, err = unprotect(data)
	if err != nil {
		return nil, fmt.Errorf("decrypting token: %w", err)
	}
	return parseToken(data)
}

// migratePlainToken converts a token stored as plain JSON by older versions
// to the encrypted file and removes the plain one.
func migratePlainToken() (*oauth2.Token, error) {
	path, err := tokenPath(plainTokenFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tok, err := parseToken(data)
	if err != nil {
		return nil, err
	}
	if err := saveToken(tok); err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return tok, nil
}

func parseToken(data []byte) (*oauth2.Token, error) {
	tok := new(oauth2.Token)
	if err := json.Unmarshal(data, tok); err != nil {
		return nil, err
	}
	return tok, nil
}

func saveToken(tok *oauth2.Token) error {
	path, err := tokenPath(tokenFile)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	data, err = protect(data)
	if err != nil {
		return fmt.Errorf("encrypting token: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func randomString() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func openBrowser(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
