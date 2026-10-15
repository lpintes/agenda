# Agenda

A simple, screen reader friendly Google Calendar viewer for Windows.

Agenda is built with [walk](https://github.com/lpintes/walk) and aims to work
well with NVDA and other screen readers: every function is reachable from the
keyboard and changes are announced.

The first version is read-only; editing events will follow.

## Status

Early development. Nothing usable yet.

## Planned features

- Day, week, month and "next 30 days" views.
- Choose which calendars are shown.
- Events list with day, time, title, calendar and location, plus a detail pane.
- Keyboard shortcuts:
  - Ctrl+Left and Ctrl+Right: previous and next period.
  - Ctrl+D: today.
  - Ctrl+G: go to date.
  - F5: refresh.
  - F6: move between panes.
  - Enter: show event detail.

## Google OAuth client

Agenda signs in to Google through your default web browser. It needs an OAuth
client of type "Desktop app" from the Google Cloud Console with the Google
Calendar API enabled.

Download the client JSON and save it as:

```
%APPDATA%\agenda\client.json
```

The client file and the stored token are never part of this repository.

## Building

Requires Go and Windows.

```
go build
```

## License

BSD 3-clause, see [LICENSE](LICENSE).
