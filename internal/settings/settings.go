// Package settings stores the user's choices between runs in
// %APPDATA%\agenda\settings.json.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lpintes/agenda/internal/auth"
)

const file = "settings.json"

// Settings are the user's choices kept between runs.
type Settings struct {
	// Range is the name of the range shown last, such as "Week".
	Range string `json:"range,omitempty"`
	// Calendars maps calendar IDs to whether their events are shown.
	// Calendars missing here use their Google Calendar setting.
	Calendars map[string]bool `json:"calendars,omitempty"`
}

// Path returns the settings file in auth.Dir.
func Path() (string, error) {
	dir, err := auth.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, file), nil
}

// Load reads the settings from path. A missing file gives empty settings.
func Load(path string) (*Settings, error) {
	s := &Settings{Calendars: map[string]bool{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return &Settings{Calendars: map[string]bool{}}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if s.Calendars == nil {
		s.Calendars = map[string]bool{}
	}
	return s, nil
}

// Save writes the settings to path. It writes a temporary file first so that
// a failed write does not destroy the previous settings.
func (s *Settings) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
