package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grunclepug/vshot/internal/config"
)

// target is the image that save and yank act on.
func (a *App) target() image.Image {
	if a.State.HasSelection() {
		return a.Image.SubImage(a.State.Selection)
	}
	return a.Image
}

// save writes the target as a PNG and returns the path written.
func (a *App) save(arg string) (string, error) {
	path, err := resolvePath(arg, time.Now())
	if err != nil {
		return "", err
	}
	// Create the default directory on first use; other paths must exist.
	if dir, err := expandHome(config.DefaultDir); err == nil && filepath.Dir(path) == filepath.Clean(dir) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return path, writePNG(path, a.target())
}

// yank copies the target to the clipboard and quits. On failure it stays
// open and shows the error.
func (a *App) yank() {
	if err := copyToClipboard(a.clipboard, a.target()); err != nil {
		a.State.Message = "Error: " + err.Error()
		return
	}
	a.quit = true
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
}

func copyToClipboard(argv []string, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	// xclip and wl-copy take ownership of the clipboard, then fork a child
	// that keeps serving it after we exit. Its stdout/stderr must not be
	// pipes, or Run would block until something else takes the clipboard.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", argv[0], err)
	}
	return nil
}

// resolvePath maps a :w argument to a file path. An empty argument yields a
// timestamped name in the default directory; relative paths are taken
// relative to the default directory.
func resolvePath(arg string, now time.Time) (string, error) {
	if arg == "~" || strings.HasPrefix(arg, "~/") {
		return expandHome(arg)
	}
	if filepath.IsAbs(arg) {
		return filepath.Clean(arg), nil
	}
	dir, err := expandHome(config.DefaultDir)
	if err != nil {
		return "", err
	}
	if arg == "" {
		arg = now.Format(config.TimestampFormat)
	}
	return filepath.Join(dir, arg), nil
}

func expandHome(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, p[1:]), nil
}
