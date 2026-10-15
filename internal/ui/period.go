package ui

import "time"

// RangeKind is the length of the period shown in the events list.
type RangeKind int

const (
	RangeDay RangeKind = iota
	RangeWeek
	RangeMonth
	Range30Days
)

// rangeNames are the range combo box items, in RangeKind order.
var rangeNames = []string{"Day", "Week", "Month", "Next 30 days"}

// Period is the half-open interval [From, To) of local days.
type Period struct {
	From, To time.Time
}

// periodOf returns the period of kind containing date. Weeks start on
// Monday; "next 30 days" starts at date itself.
func periodOf(kind RangeKind, date time.Time) Period {
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	switch kind {
	case RangeWeek:
		from := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
		return Period{from, from.AddDate(0, 0, 7)}
	case RangeMonth:
		from := day.AddDate(0, 0, 1-day.Day())
		return Period{from, from.AddDate(0, 1, 0)}
	case Range30Days:
		return Period{day, day.AddDate(0, 0, 30)}
	default:
		return Period{day, day.AddDate(0, 0, 1)}
	}
}

// shift moves date by n periods of kind.
func shift(kind RangeKind, date time.Time, n int) time.Time {
	switch kind {
	case RangeWeek:
		return date.AddDate(0, 0, 7*n)
	case RangeMonth:
		// Clamp to the last day so that 31 January moves to February.
		first := time.Date(date.Year(), date.Month()+time.Month(n), 1, 0, 0, 0, 0, date.Location())
		last := first.AddDate(0, 1, -1).Day()
		return first.AddDate(0, 0, min(date.Day(), last)-1)
	case Range30Days:
		return date.AddDate(0, 0, 30*n)
	default:
		return date.AddDate(0, 0, n)
	}
}

// contains reports whether t falls in the period.
func (p Period) contains(t time.Time) bool {
	return !t.Before(p.From) && t.Before(p.To)
}

// String describes the period for the status bar, for example
// "Mon 12 Oct 2026 to Sun 18 Oct 2026".
func (p Period) String() string {
	const layout = "Mon 2 Jan 2006"
	last := p.To.AddDate(0, 0, -1)
	if last.Equal(p.From) {
		return p.From.Format(layout)
	}
	return p.From.Format(layout) + " to " + last.Format(layout)
}
