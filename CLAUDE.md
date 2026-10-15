# Agenda

Screen reader friendly Google Calendar viewer for Windows, written in Go with
the GUI library `github.com/lpintes/walk` (local clone in
`C:\vcs\github.com\lpintes\walk`). Read-only first, editing later.

## Layout

- `internal/auth`: OAuth 2.0 for installed apps (system browser, loopback
  redirect to `127.0.0.1:<random port>`, PKCE, state). Embedded webviews are
  blocked by Google, do not use them.
- `internal/calendar`: Google Calendar API wrapper with the app's own
  `Calendar` and `Event` model; recurring events are expanded
  (`singleEvents=true`).
- `cmd/agenda-check`: console check that signs in and prints the next seven
  days. Use it to verify auth and API changes without the GUI.
- `internal/ui` and `main.go`: GUI. `period.go` holds the date logic
  (ranges, shifting, row to select) and `details.go` the text of the detail
  pane; both are unit tested. `ui.go` is the main window.

## Build and run

```
./build.sh -test
.\agenda.exe
.\agenda-check.exe
```

`build.sh` builds both executables; `-test` first runs `go vet` and
`go test`. `rsrc_windows_amd64.syso` is committed; regenerate it with
`go generate .` after changing `agenda.exe.manifest`.

The first run opens the default browser for Google sign-in; the user must
sign in and allow access (the "unverified app" warning needs "Continue").
Later runs reuse the stored token without the browser. Before running
anything that opens a browser or a window, tell the user what will appear
and what they must do.

## OAuth client and token

- The client JSON (Desktop app type) lives in `%APPDATA%\agenda\client.json`
  and is never committed. Do not print its contents.
- The token is stored in `%APPDATA%\agenda\token.json`, for now as plain
  JSON. Planned: Windows Credential Manager or DPAPI.
- Scopes: `calendar.readonly` now, `calendar.events` once editing exists.
- The OAuth consent screen is in Testing mode, where refresh tokens expire
  after 7 days. An `invalid_grant` error usually means this; delete the
  token and sign in again.
- The repository is public: never commit the Cloud project ID, e-mail
  addresses or anything from the client file.

## GUI (walk)

- walk needs a manifest enabling comctl32 v6, embedded as `rsrc.syso`
  (see `walk/examples/*/rsrc.syso`, built with
  `github.com/akavel/rsrc`). The examples ship 386 COFF; generate one per
  target architecture.
- walk has no month calendar widget, only `DateEdit`.
- walk is used from GitHub (`go get github.com/lpintes/walk@<commit>`).
  To test a walk fix first, use a temporary
  `go mod edit -replace github.com/lpintes/walk=../walk` and drop it
  before committing. `gh` in the walk clone needs `--repo lpintes/walk`
  (it also has the `lxn/walk` upstream remote).

## Accessibility conventions

The user is blind and works with NVDA.

- Everything must be reachable from the keyboard, with a sensible Tab order:
  fields the user works with first, rarely used buttons last.
- Do not announce routine results such as "5 events": with NVDA the
  announcement talks over the control being read, even with
  `AnnounceMostRecent`. Put them in the status bar; announce only errors.
- Open question: `DateEdit` (the native date picker) is hard to use with
  NVDA: it does not say whether day, month or year is focused, and the Up
  arrow moves forward. Look for a better way to pick a date.
- Planned shortcuts: Ctrl+Left and Ctrl+Right previous and next period,
  Ctrl+D today, Ctrl+G go to date, F5 refresh, F6 cycle panes, Enter detail;
  later Ctrl+N new, F2 edit, Delete delete.

## Conventions

- UI strings in English, prepared for translation.
- Commit straight to `main` unless told otherwise.
