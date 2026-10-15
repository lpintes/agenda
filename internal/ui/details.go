package ui

import (
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/lpintes/agenda/internal/calendar"
)

// details is the text of the detail pane for e, with Windows line endings.
func details(e calendar.Event) string {
	lines := []string{e.Title, "When: " + when(e), "Calendar: " + e.Calendar}
	if e.Location != "" {
		lines = append(lines, "Location: "+e.Location)
	}
	if d := plainText(e.Description); d != "" {
		lines = append(lines, "", d)
	}
	return strings.Join(lines, "\r\n")
}

const dateLayout = "Mon 2 Jan 2006"

// when describes the time of e; the end of all-day events is exclusive.
func when(e calendar.Event) string {
	if e.AllDay {
		last := e.End.AddDate(0, 0, -1)
		if !last.After(e.Start) {
			return e.Start.Format(dateLayout) + ", all day"
		}
		return e.Start.Format(dateLayout) + " to " + last.Format(dateLayout) + ", all day"
	}
	if sameDay(e.Start, e.End) {
		return e.Start.Format(dateLayout) + ", " + e.Start.Format("15:04") + " to " + e.End.Format("15:04")
	}
	return e.Start.Format(dateLayout+", 15:04") + " to " + e.End.Format(dateLayout+", 15:04")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

var (
	lineBreakTag = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</li>`)
	anyTag       = regexp.MustCompile(`<[^>]*>`)
	blankLines   = regexp.MustCompile(`\n{3,}`)
)

// plainText converts an event description, which Google Calendar may store
// as simple HTML, to plain text with Windows line endings.
func plainText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = lineBreakTag.ReplaceAllString(s, "\n")
	s = anyTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = blankLines.ReplaceAllString(strings.TrimSpace(s), "\n\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}
