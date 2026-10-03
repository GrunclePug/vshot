package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grunclepug/vshot/internal/config"
)

func TestParseCommand(t *testing.T) {
	for _, tc := range []struct{ line, cmd, arg string }{
		{"w", "w", ""},
		{"  wq  out.png ", "wq", "out.png"},
		{"w my shot.png", "w", "my shot.png"},
		{"wat", "wat", ""},
		{"", "", ""},
	} {
		cmd, arg := parseCommand(tc.line)
		if cmd != tc.cmd || arg != tc.arg {
			t.Errorf("parseCommand(%q) = %q, %q; want %q, %q", tc.line, cmd, arg, tc.cmd, tc.arg)
		}
	}
}

func TestUnknownCommandDoesNotSave(t *testing.T) {
	a := &App{State: NewState(bounds)}
	a.execute("wat")
	if a.quit || a.State.Message != "Not an editor command: wat" {
		t.Fatalf("quit=%v message=%q", a.quit, a.State.Message)
	}
}

func TestResolvePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	dir, _ := expandHome(config.DefaultDir)
	now := time.Date(2026, 10, 3, 14, 5, 9, 0, time.UTC)

	for _, tc := range []struct{ arg, want string }{
		{"", filepath.Join(dir, now.Format(config.TimestampFormat))},
		{"shot.png", filepath.Join(dir, "shot.png")},
		{"~/shot.png", filepath.Join(home, "shot.png")},
		{"/tmp/shot.png", "/tmp/shot.png"},
		{"~other/shot.png", filepath.Join(dir, "~other/shot.png")},
	} {
		got, err := resolvePath(tc.arg, now)
		if err != nil || got != tc.want {
			t.Errorf("resolvePath(%q) = %q, %v; want %q", tc.arg, got, err, tc.want)
		}
	}
}
