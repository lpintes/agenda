// Command agenda is a screen reader friendly Google Calendar viewer.
package main

import (
	"log"

	"github.com/lpintes/agenda/internal/ui"
)

//go:generate go run github.com/akavel/rsrc@v0.10.2 -manifest agenda.exe.manifest -arch amd64 -o rsrc_windows_amd64.syso

func main() {
	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
