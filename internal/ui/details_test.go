package ui

import (
	"testing"
	"time"

	"github.com/lpintes/agenda/internal/calendar"
)

func TestWhen(t *testing.T) {
	at := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, time.Local) }
	tests := []struct {
		name string
		e    calendar.Event
		want string
	}{
		{"timed", calendar.Event{Start: at(15, 10, 0), End: at(15, 11, 30)},
			"Thu 15 Oct 2026, 10:00 to 11:30"},
		{"overnight", calendar.Event{Start: at(15, 22, 0), End: at(16, 2, 0)},
			"Thu 15 Oct 2026, 22:00 to Fri 16 Oct 2026, 02:00"},
		{"one day", calendar.Event{AllDay: true, Start: at(15, 0, 0), End: at(16, 0, 0)},
			"Thu 15 Oct 2026, all day"},
		{"three days", calendar.Event{AllDay: true, Start: at(15, 0, 0), End: at(18, 0, 0)},
			"Thu 15 Oct 2026 to Sat 17 Oct 2026, all day"},
	}
	for _, tt := range tests {
		if got := when(tt.e); got != tt.want {
			t.Errorf("%s: when() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPlainText(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"Plain\ntext", "Plain\r\ntext"},
		{"Line<br>next<BR/>last", "Line\r\nnext\r\nlast"},
		{`See <a href="https://example.com">site</a> &amp; more`, "See site & more"},
		{"<p>One</p><p>Two</p>\n\n\n\nThree", "One\r\nTwo\r\n\r\nThree"},
	}
	for _, tt := range tests {
		if got := plainText(tt.in); got != tt.want {
			t.Errorf("plainText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDetails(t *testing.T) {
	e := calendar.Event{
		Title:    "Meeting",
		Calendar: "Work",
		Start:    time.Date(2026, 10, 15, 9, 0, 0, 0, time.Local),
		End:      time.Date(2026, 10, 15, 10, 0, 0, 0, time.Local),
	}
	want := "Meeting\r\nWhen: Thu 15 Oct 2026, 09:00 to 10:00\r\nCalendar: Work"
	if got := details(e); got != want {
		t.Errorf("details() = %q, want %q", got, want)
	}
	e.Location, e.Description = "Room 1", "Agenda"
	want += "\r\nLocation: Room 1\r\n\r\nAgenda"
	if got := details(e); got != want {
		t.Errorf("details() = %q, want %q", got, want)
	}
}
