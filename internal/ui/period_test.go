package ui

import (
	"testing"
	"time"

	"github.com/lpintes/agenda/internal/calendar"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func TestPeriodOf(t *testing.T) {
	thu := time.Date(2026, 10, 15, 14, 30, 0, 0, time.Local)
	tests := []struct {
		kind     RangeKind
		from, to time.Time
	}{
		{RangeDay, date(2026, 10, 15), date(2026, 10, 16)},
		{RangeWeek, date(2026, 10, 12), date(2026, 10, 19)},
		{RangeMonth, date(2026, 10, 1), date(2026, 11, 1)},
		{Range30Days, date(2026, 10, 15), date(2026, 11, 14)},
	}
	for _, tt := range tests {
		p := periodOf(tt.kind, thu)
		if !p.From.Equal(tt.from) || !p.To.Equal(tt.to) {
			t.Errorf("%s: got %v to %v, want %v to %v", rangeNames[tt.kind], p.From, p.To, tt.from, tt.to)
		}
	}
	// Sunday belongs to the week starting the Monday before.
	if p := periodOf(RangeWeek, date(2026, 10, 18)); !p.From.Equal(date(2026, 10, 12)) {
		t.Errorf("week of Sunday starts %v", p.From)
	}
}

func TestShift(t *testing.T) {
	tests := []struct {
		kind RangeKind
		from time.Time
		n    int
		want time.Time
	}{
		{RangeDay, date(2026, 12, 31), 1, date(2027, 1, 1)},
		{RangeWeek, date(2026, 10, 15), -1, date(2026, 10, 8)},
		{RangeMonth, date(2026, 1, 31), 1, date(2026, 2, 28)},
		{RangeMonth, date(2026, 3, 31), -1, date(2026, 2, 28)},
		{RangeMonth, date(2026, 12, 15), 1, date(2027, 1, 15)},
		{Range30Days, date(2026, 10, 15), 1, date(2026, 11, 14)},
	}
	for _, tt := range tests {
		if got := shift(tt.kind, tt.from, tt.n); !got.Equal(tt.want) {
			t.Errorf("%s %v %+d: got %v, want %v", rangeNames[tt.kind], tt.from, tt.n, got, tt.want)
		}
	}
}

func TestFirstRow(t *testing.T) {
	at := func(d, h int) time.Time { return time.Date(2026, 10, d, h, 0, 0, 0, time.Local) }
	events := []calendar.Event{
		{Start: at(14, 9), End: at(14, 10)},
		{Start: at(15, 0), End: at(16, 0), AllDay: true},
		{Start: at(15, 9), End: at(15, 10)},
		{Start: at(15, 15), End: at(15, 16)},
		{Start: at(17, 9), End: at(17, 10)},
	}
	week := periodOf(RangeWeek, at(15, 0))
	tests := []struct {
		name   string
		period Period
		now    time.Time
		want   int
	}{
		{"today's all-day event first", week, at(15, 12), 1},
		{"after today's events", week, at(16, 12), 4},
		{"after all events", week, at(18, 12), 4},
		{"before all events", week, at(12, 8), 0},
		{"another week", periodOf(RangeWeek, at(22, 0)), at(15, 12), 0},
	}
	for _, tt := range tests {
		if got := firstRow(events, tt.period, tt.now); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestPeriodString(t *testing.T) {
	if got := periodOf(RangeDay, date(2026, 10, 15)).String(); got != "Thu 15 Oct 2026" {
		t.Errorf("day: %q", got)
	}
	if got := periodOf(RangeWeek, date(2026, 10, 15)).String(); got != "Mon 12 Oct 2026 to Sun 18 Oct 2026" {
		t.Errorf("week: %q", got)
	}
}
