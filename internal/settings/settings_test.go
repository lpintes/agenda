package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Range != "" || s.Calendars == nil || len(s.Calendars) != 0 {
		t.Errorf("got %+v, want empty settings", s)
	}
}

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "settings.json")
	want := &Settings{Range: "Month", Calendars: map[string]bool{"a": true, "b": false}}
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	// Saving again replaces the existing file.
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadBroken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err == nil {
		t.Error("no error for a broken file")
	}
	if s == nil || s.Calendars == nil {
		t.Errorf("got %+v, want usable empty settings", s)
	}
}
