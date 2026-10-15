// Package ui is the main window of Agenda.
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lpintes/walk"
	. "github.com/lpintes/walk/declarative"

	"github.com/lpintes/agenda/internal/auth"
	"github.com/lpintes/agenda/internal/calendar"
)

type app struct {
	mw       *walk.MainWindow
	dateEdit *walk.DateEdit
	rangeBox *walk.ComboBox
	table    *walk.TableView
	status   *walk.StatusBarItem
	events   *eventModel

	// Owned by the GUI thread.
	svc  *calendar.Service
	cals []calendar.Calendar // calendars whose events are shown
	seq  int                 // number of the latest events request
}

// Run shows the main window and returns when it is closed.
func Run() error {
	a := &app{events: new(eventModel)}

	err := MainWindow{
		AssignTo: &a.mw,
		Title:    "Agenda",
		MinSize:  Size{Width: 640, Height: 480},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					Label{Text: "&Date:"},
					DateEdit{
						AssignTo:      &a.dateEdit,
						Date:          time.Now(),
						OnDateChanged: a.reload,
					},
					Label{Text: "&Range:"},
					ComboBox{
						AssignTo:              &a.rangeBox,
						Model:                 rangeNames,
						CurrentIndex:          int(RangeWeek),
						OnCurrentIndexChanged: a.reload,
					},
					HSpacer{},
				},
			},
			Label{Text: "&Events:"},
			TableView{
				AssignTo:                 &a.table,
				Model:                    a.events,
				NotSortableByHeaderClick: true,
				LastColumnStretched:      true,
				Columns: []TableViewColumn{
					{Title: "Day", Width: 110},
					{Title: "Time", Width: 110},
					{Title: "Title", Width: 250},
					{Title: "Calendar", Width: 120},
					{Title: "Location"},
				},
			},
			// Below the table so that Tab reaches the events first.
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{Text: "&Previous", OnClicked: func() { a.move(-1) }},
					PushButton{Text: "&Today", OnClicked: a.today},
					PushButton{Text: "&Next", OnClicked: func() { a.move(1) }},
					HSpacer{},
				},
			},
		},
		StatusBarItems: []StatusBarItem{
			{AssignTo: &a.status, Text: "Signing in..."},
		},
	}.Create()
	if err != nil {
		return err
	}
	go a.connect()
	a.mw.Run()
	return nil
}

func (a *app) rangeKind() RangeKind {
	return RangeKind(max(a.rangeBox.CurrentIndex(), 0))
}

func (a *app) move(n int) {
	a.dateEdit.SetDate(shift(a.rangeKind(), a.dateEdit.Date(), n))
}

func (a *app) today() {
	a.dateEdit.SetDate(time.Now())
}

// connect signs in and reads the calendar list in the background; signing
// in may open the browser the first time.
func (a *app) connect() {
	ctx := context.Background()
	svc, cals, err := func() (*calendar.Service, []calendar.Calendar, error) {
		cfg, err := auth.LoadConfig(calendar.ReadOnlyScope)
		if err != nil {
			return nil, nil, err
		}
		ts, err := auth.TokenSource(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		svc, err := calendar.New(ctx, ts)
		if err != nil {
			return nil, nil, err
		}
		all, err := svc.Calendars(ctx)
		if err != nil {
			return nil, nil, err
		}
		var shown []calendar.Calendar
		for _, c := range all {
			if c.Selected {
				shown = append(shown, c)
			}
		}
		return svc, shown, nil
	}()
	a.mw.Synchronize(func() {
		if err != nil {
			a.report(fmt.Sprintf("Sign-in failed: %v", err))
			return
		}
		a.svc, a.cals = svc, cals
		a.reload()
	})
}

// reloadDelay lets quick changes, such as arrow keys in the date field,
// settle before the events are read.
const reloadDelay = 300 * time.Millisecond

// reload reads the events of the current period in the background. Results
// of requests superseded by a later one are dropped.
func (a *app) reload() {
	if a.svc == nil || a.rangeBox == nil {
		return
	}
	a.seq++
	seq := a.seq
	time.AfterFunc(reloadDelay, func() {
		a.mw.Synchronize(func() {
			if seq == a.seq {
				a.fetch(seq)
			}
		})
	})
}

func (a *app) fetch(seq int) {
	period := periodOf(a.rangeKind(), a.dateEdit.Date())
	svc, cals := a.svc, a.cals
	a.status.SetText("Loading " + period.String() + "...")
	go func() {
		events, err := svc.Events(context.Background(), cals, period.From, period.To)
		a.mw.Synchronize(func() {
			if seq != a.seq {
				return
			}
			if err != nil {
				a.report(fmt.Sprintf("Loading events failed: %v", err))
				return
			}
			a.events.set(events)
			if len(events) > 0 {
				a.table.SetCurrentIndex(firstRow(events, period, time.Now()))
			}
			// Not announced: it would talk over the field the user is in.
			a.status.SetText(fmt.Sprintf("%s, %s", capitalize(countEvents(len(events))), period))
		})
	}()
}

// firstRow is the row to select after loading: the first event that has not
// ended yet if the period contains now, otherwise the first event.
func firstRow(events []calendar.Event, period Period, now time.Time) int {
	if !period.contains(now) {
		return 0
	}
	for i, e := range events {
		if e.End.After(now) {
			return i
		}
	}
	return len(events) - 1
}

// report shows an error in the status bar and has screen readers speak it
// after what they are saying now.
func (a *app) report(text string) {
	a.status.SetText(text)
	walk.Announce(a.mw, text, walk.AnnounceMostRecent)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func countEvents(n int) string {
	switch n {
	case 0:
		return "no events"
	case 1:
		return "1 event"
	default:
		return fmt.Sprintf("%d events", n)
	}
}

// eventModel is the table model of the events list.
type eventModel struct {
	walk.TableModelBase
	events []calendar.Event
}

func (m *eventModel) set(events []calendar.Event) {
	m.events = events
	m.PublishRowsReset()
}

func (m *eventModel) RowCount() int {
	return len(m.events)
}

func (m *eventModel) Value(row, col int) any {
	e := m.events[row]
	switch col {
	case 0:
		return e.Start.Format("Mon 2 Jan")
	case 1:
		if e.AllDay {
			return "all day"
		}
		return e.Start.Format("15:04") + " to " + e.End.Format("15:04")
	case 2:
		return e.Title
	case 3:
		return e.Calendar
	case 4:
		return e.Location
	}
	return ""
}
