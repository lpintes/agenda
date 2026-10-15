// Package calendar reads calendars and events from Google Calendar and
// converts them to the application's own model.
package calendar

import (
	"context"
	"fmt"
	"sort"
	"time"

	"golang.org/x/oauth2"
	gcal "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// ReadOnlyScope is the OAuth scope needed for reading.
const ReadOnlyScope = gcal.CalendarReadonlyScope

// Calendar is one calendar from the user's calendar list.
type Calendar struct {
	ID       string
	Name     string
	Primary  bool
	Selected bool // shown in the Google Calendar web interface
}

// Event is a single event occurrence; recurring events are expanded.
type Event struct {
	ID          string
	CalendarID  string
	Calendar    string
	Title       string
	Start       time.Time
	End         time.Time
	AllDay      bool
	Location    string
	Description string
	Link        string
}

// Service wraps the Google Calendar API.
type Service struct {
	api *gcal.Service
}

// New creates a service authorized by ts.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	api, err := gcal.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, err
	}
	return &Service{api: api}, nil
}

// Calendars returns the user's calendar list, primary calendar first.
func (s *Service) Calendars(ctx context.Context) ([]Calendar, error) {
	var cals []Calendar
	err := s.api.CalendarList.List().Context(ctx).Pages(ctx, func(page *gcal.CalendarList) error {
		for _, c := range page.Items {
			name := c.SummaryOverride
			if name == "" {
				name = c.Summary
			}
			cals = append(cals, Calendar{ID: c.Id, Name: name, Primary: c.Primary, Selected: c.Selected})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(cals, func(i, j int) bool { return cals[i].Primary && !cals[j].Primary })
	return cals, nil
}

// Events returns the events of cals overlapping [from, to), sorted by start.
func (s *Service) Events(ctx context.Context, cals []Calendar, from, to time.Time) ([]Event, error) {
	var events []Event
	for _, c := range cals {
		call := s.api.Events.List(c.ID).Context(ctx).
			SingleEvents(true).
			OrderBy("startTime").
			TimeMin(from.Format(time.RFC3339)).
			TimeMax(to.Format(time.RFC3339))
		err := call.Pages(ctx, func(page *gcal.Events) error {
			for _, item := range page.Items {
				if item.Status == "cancelled" {
					continue
				}
				e, err := convert(item, c, from.Location())
				if err != nil {
					return err
				}
				events = append(events, e)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("calendar %s: %w", c.Name, err)
		}
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Start.Before(events[j].Start) })
	return events, nil
}

func convert(item *gcal.Event, c Calendar, loc *time.Location) (Event, error) {
	e := Event{
		ID:          item.Id,
		CalendarID:  c.ID,
		Calendar:    c.Name,
		Title:       item.Summary,
		Location:    item.Location,
		Description: item.Description,
		Link:        item.HtmlLink,
	}
	var err error
	if e.Start, e.AllDay, err = parseTime(item.Start, loc); err != nil {
		return e, err
	}
	if e.End, _, err = parseTime(item.End, loc); err != nil {
		return e, err
	}
	return e, nil
}

// parseTime converts an API time; all-day dates become local midnight.
func parseTime(t *gcal.EventDateTime, loc *time.Location) (time.Time, bool, error) {
	if t == nil {
		return time.Time{}, false, nil
	}
	if t.DateTime != "" {
		v, err := time.Parse(time.RFC3339, t.DateTime)
		return v.In(loc), false, err
	}
	v, err := time.ParseInLocation(time.DateOnly, t.Date, loc)
	return v, true, err
}
