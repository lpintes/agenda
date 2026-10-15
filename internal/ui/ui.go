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
	detail   *walk.TextEdit
	calTable *walk.TableView
	buttons  *walk.Composite
	status   *walk.StatusBarItem
	events   *eventModel
	cals     *calendarModel

	// Owned by the GUI thread.
	svc        *calendar.Service
	connecting bool
	seq        int // number of the latest events request
}

// Run shows the main window and returns when it is closed.
func Run() error {
	a := &app{events: new(eventModel)}
	a.cals = &calendarModel{onChange: a.reload}

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
				OnCurrentIndexChanged:    a.showDetail,
				Columns: []TableViewColumn{
					{Title: "Day", Width: 110},
					{Title: "Time", Width: 110},
					{Title: "Title", Width: 250},
					{Title: "Calendar", Width: 120},
					{Title: "Location"},
				},
			},
			Label{Text: "Deta&ils:"},
			TextEdit{
				AssignTo: &a.detail,
				ReadOnly: true,
				VScroll:  true,
				MinSize:  Size{Height: 100},
				MaxSize:  Size{Height: 160},
			},
			Label{Text: "&Calendars:"},
			TableView{
				AssignTo:                 &a.calTable,
				Model:                    a.cals,
				CheckBoxes:               true,
				HeaderHidden:             true,
				NotSortableByHeaderClick: true,
				LastColumnStretched:      true,
				MinSize:                  Size{Height: 60},
				MaxSize:                  Size{Height: 100},
				Columns:                  []TableViewColumn{{Title: "Calendar"}},
			},
			// Last in the Tab order: rarely used.
			Composite{
				AssignTo: &a.buttons,
				Layout:   HBox{MarginsZero: true},
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
	if err := a.addShortcuts(); err != nil {
		return err
	}
	a.startConnect()
	a.mw.Run()
	return nil
}

// addShortcuts sets up the keyboard shortcuts of the main window.
func (a *app) addShortcuts() error {
	// Not in the detail pane, where Ctrl+Left and Ctrl+Right move by words.
	period := []*walk.Action{
		shortcut(walk.ModControl, walk.KeyLeft, func() { a.move(-1) }),
		shortcut(walk.ModControl, walk.KeyRight, func() { a.move(1) }),
	}
	for _, w := range []walk.Window{a.dateEdit, a.rangeBox, a.table, a.calTable, a.buttons} {
		for _, action := range period {
			if err := w.AsWindowBase().ShortcutActions().Add(action); err != nil {
				return err
			}
		}
	}
	global := []*walk.Action{
		shortcut(walk.ModControl, walk.KeyD, a.today),
		shortcut(walk.ModControl, walk.KeyG, func() { a.dateEdit.SetFocus() }),
		shortcut(0, walk.KeyF5, a.refresh),
		shortcut(0, walk.KeyF6, func() { a.cyclePanes(1) }),
		shortcut(walk.ModShift, walk.KeyF6, func() { a.cyclePanes(-1) }),
	}
	for _, action := range global {
		if err := a.mw.ShortcutActions().Add(action); err != nil {
			return err
		}
	}
	return nil
}

func shortcut(mods walk.Modifiers, key walk.Key, f func()) *walk.Action {
	action := walk.NewAction()
	action.SetShortcut(walk.Shortcut{Modifiers: mods, Key: key})
	action.Triggered().Attach(f)
	return action
}

// cyclePanes moves the focus by n panes: date, events, details, calendars.
func (a *app) cyclePanes(n int) {
	panes := []walk.Widget{a.dateEdit, a.table, a.detail, a.calTable}
	current := -1
	if a.rangeBox.Focused() {
		current = 0 // the range belongs to the date pane
	}
	for i, p := range panes {
		if p.Focused() {
			current = i
		}
	}
	next := (current + n + len(panes)) % len(panes)
	if current < 0 && n < 0 {
		next = len(panes) - 1
	}
	panes[next].SetFocus()
}

// refresh reads the events again, or retries signing in if it failed.
func (a *app) refresh() {
	if a.svc == nil {
		a.startConnect()
		return
	}
	a.reload()
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

// startConnect starts connect unless it is already running.
func (a *app) startConnect() {
	if a.connecting {
		return
	}
	a.connecting = true
	a.status.SetText("Signing in...")
	go a.connect()
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
		cals, err := svc.Calendars(ctx)
		if err != nil {
			return nil, nil, err
		}
		return svc, cals, nil
	}()
	a.mw.Synchronize(func() {
		a.connecting = false
		if err != nil {
			a.report(fmt.Sprintf("Sign-in failed: %v. Press F5 to try again.", err))
			return
		}
		a.svc = svc
		a.cals.set(cals)
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
	svc, cals := a.svc, a.cals.shown()
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
			a.showDetail()
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

// showDetail shows the current event in the detail pane.
func (a *app) showDetail() {
	if a.detail == nil {
		return
	}
	text := ""
	if i := a.table.CurrentIndex(); i >= 0 && i < len(a.events.events) {
		text = details(a.events.events[i])
	}
	a.detail.SetText(text)
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

// calendarModel is the model of the checkable calendar list.
type calendarModel struct {
	walk.TableModelBase
	cals     []calendar.Calendar
	checked  []bool
	onChange func() // called when a calendar is checked or unchecked
}

// set fills the list; calendars shown in Google Calendar start checked.
func (m *calendarModel) set(cals []calendar.Calendar) {
	m.cals = cals
	m.checked = make([]bool, len(cals))
	for i, c := range cals {
		m.checked[i] = c.Selected
	}
	m.PublishRowsReset()
}

// shown returns the checked calendars.
func (m *calendarModel) shown() []calendar.Calendar {
	var shown []calendar.Calendar
	for i, c := range m.cals {
		if m.checked[i] {
			shown = append(shown, c)
		}
	}
	return shown
}

func (m *calendarModel) RowCount() int {
	return len(m.cals)
}

func (m *calendarModel) Value(row, col int) any {
	return m.cals[row].Name
}

func (m *calendarModel) Checked(row int) bool {
	return m.checked[row]
}

func (m *calendarModel) SetChecked(row int, checked bool) error {
	m.checked[row] = checked
	m.onChange()
	return nil
}
