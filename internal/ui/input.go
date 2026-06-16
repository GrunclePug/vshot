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
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func HandleKeyPress(X *xgb.Conn, e xproto.KeyPressEvent, state *State, redraw func()) {
	shift := (e.State & 1) != 0
	switch state.Mode {
	case ModeCommand:
		handleCommandKeys(X, e.Detail, shift, state, redraw)
	case ModeVisual:
		handleVisualKeys(X, e.Detail, state, redraw)
	default:
		handleNormalKeys(X, e.Detail, state, redraw)
	}
}

func handleNormalKeys(X *xgb.Conn, key xproto.Keycode, state *State, redraw func()) {
	switch key {
	case 9: // Escape
		cleanupAndExit(X)
	case 47: // ':'
		state.Mode = ModeCommand
		state.CommandBuffer = ""
		state.ErrorMessage = ""
		redraw()
	case 29: // 'y'
		yankToClipboard(state)
		state.ClearSelection()
		redraw()
	}
}

func handleVisualKeys(X *xgb.Conn, key xproto.Keycode, state *State, redraw func()) {
	switch key {
	case 9: // Escape
		state.ClearSelection()
		redraw()
	case 47: // ':'
		state.Mode = ModeCommand
		state.CommandBuffer = ""
		state.ErrorMessage = ""
		redraw()
	case 29: // 'y'
		yankToClipboard(state)
		state.ClearSelection()
		redraw()
	}
}

func handleCommandKeys(X *xgb.Conn, key xproto.Keycode, shift bool, state *State, redraw func()) {
	switch key {
	case 9: // Escape
		state.Mode = ModeNormal
		state.CommandBuffer = ""
		redraw()
	case 36: // Enter
		executeCommand(X, state, redraw)
	case 22: // Backspace
		if len(state.CommandBuffer) > 0 {
			state.CommandBuffer = state.CommandBuffer[:len(state.CommandBuffer)-1]
		} else {
			state.Mode = ModeNormal
		}
		redraw()
	default:
		char := translateKeycode(key, shift)
		if char != "" {
			state.CommandBuffer += char
			redraw()
		}
	}
}

func executeCommand(X *xgb.Conn, state *State, redraw func()) {
	cmd := strings.TrimSpace(state.CommandBuffer)
	state.Mode = ModeNormal
	state.CommandBuffer = ""

	if cmd == "q" {
		cleanupAndExit(X)
	} else if cmd == "wq" || strings.HasPrefix(cmd, "w") {
		args := strings.Fields(cmd)
		path := resolvePath("")
		if len(args) > 1 {
			path = resolvePath(args[1])
		}

		if err := saveScreenshot(state, path); err != nil {
			state.ErrorMessage = err.Error()
		} else {
			state.ErrorMessage = fmt.Sprintf("Saved to %s", filepath.Base(path))
		}

		if cmd == "wq" || strings.HasPrefix(cmd, "wq") {
			cleanupAndExit(X)
		}
	} else {
		state.ErrorMessage = "Not an editor command"
	}
	redraw()
}

func saveScreenshot(state *State, path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("Error: Directory does not exist")
	}

	var img image.Image
	if state.HasSelection {
		img = CropImage(state.Screenshot.Image, state.Selection)
	} else {
		img = state.Screenshot.Image
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Error: Could not create file")
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("Error: PNG encoding failed")
	}
	return nil
}

func yankToClipboard(state *State) {
	var img image.Image
	if state.HasSelection {
		img = CropImage(state.Screenshot.Image, state.Selection)
		state.ErrorMessage = "Yanked selection to clipboard!"
	} else {
		img = state.Screenshot.Image
		state.ErrorMessage = "Yanked full screen to clipboard!"
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png")
		cmd.Stdin = &buf
		if err := cmd.Run(); err != nil {
			cmdSel := exec.Command("xsel", "--clipboard", "--input")
			cmdSel.Stdin = &buf
			_ = cmdSel.Run()
		}
	}
}

func cleanupAndExit(X *xgb.Conn) {
	xproto.UngrabKeyboard(X, xproto.TimeCurrentTime)
	xproto.UngrabPointer(X, xproto.TimeCurrentTime)
	os.Exit(0)
}

func resolvePath(input string) string {
	home, _ := os.UserHomeDir()
	defaultDir := config.DefaultDir
	if strings.HasPrefix(defaultDir, "~") {
		defaultDir = filepath.Join(home, defaultDir[1:])
	}
	if input == "" {
		return filepath.Join(defaultDir, fmt.Sprintf("Screenshot_%s.png", time.Now().Format("2006-01-02_150405")))
	}
	if strings.HasPrefix(input, "~") {
		return filepath.Join(home, input[1:])
	}
	if filepath.IsAbs(input) {
		return input
	}
	return filepath.Join(defaultDir, input)
}

func translateKeycode(code xproto.Keycode, shift bool) string {
	maps := map[xproto.Keycode]string{
		10: "1", 11: "2", 12: "3", 13: "4", 14: "5", 15: "6", 16: "7", 17: "8", 18: "9", 19: "0", 20: "-", 21: "=",
		24: "q", 25: "w", 26: "e", 27: "r", 28: "t", 29: "y", 30: "u", 31: "i", 32: "o", 33: "p",
		38: "a", 39: "s", 40: "d", 41: "f", 42: "g", 43: "h", 44: "j", 45: "k", 46: "l", 47: ";", 48: "'",
		52: "z", 53: "x", 54: "c", 55: "v", 56: "b", 57: "n", 58: "m", 59: ",", 60: ".", 61: "/",
		49: "`", 65: " ",
	}
	if shift && code == 49 {
		return "~"
	}
	return maps[code]
}
