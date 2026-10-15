// Command agenda-check signs in to Google and prints the events of the next
// seven days. It verifies authentication without the GUI.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lpintes/agenda/internal/auth"
	"github.com/lpintes/agenda/internal/calendar"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := auth.LoadConfig(calendar.ReadOnlyScope)
	if err != nil {
		return err
	}
	ts, err := auth.TokenSource(ctx, cfg)
	if err != nil {
		return err
	}
	svc, err := calendar.New(ctx, ts)
	if err != nil {
		return err
	}

	cals, err := svc.Calendars(ctx)
	if err != nil {
		return err
	}
	var shown []calendar.Calendar
	fmt.Println("Calendars:")
	for _, c := range cals {
		mark := ""
		if c.Selected {
			shown = append(shown, c)
			mark = ", shown"
		}
		fmt.Printf("  %s%s\n", c.Name, mark)
	}

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 0, 7)
	events, err := svc.Events(ctx, shown, from, to)
	if err != nil {
		return err
	}
	fmt.Printf("\nEvents from %s to %s: %d\n", from.Format("Mon 2 Jan"), to.AddDate(0, 0, -1).Format("Mon 2 Jan"), len(events))
	for _, e := range events {
		when := "all day"
		if !e.AllDay {
			when = e.Start.Format("15:04") + " to " + e.End.Format("15:04")
		}
		line := fmt.Sprintf("  %s, %s: %s (%s)", e.Start.Format("Mon 2 Jan"), when, e.Title, e.Calendar)
		if e.Location != "" {
			line += ", " + e.Location
		}
		fmt.Println(line)
	}
	return nil
}
