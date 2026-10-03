package ui

import (
	"fmt"
	"strings"
)

// Key handles a key press, translated to a keysym by the backend.
func (a *App) Key(sym Keysym) {
	st := a.State
	if st.Command {
		a.commandKey(sym)
		return
	}

	switch sym {
	case KeyEscape:
		if st.HasSelection() {
			st.ClearSelection()
			st.Message = ""
		} else {
			a.quit = true
		}
	case ':':
		st.Command, st.CommandBuf, st.Message = true, "", ""
	case 'y':
		a.yank()
	}
}

func (a *App) commandKey(sym Keysym) {
	st := a.State
	switch {
	case sym == KeyEscape:
		st.Command = false
	case sym == KeyReturn || sym == KeyKPEnter:
		st.Command = false
		a.execute(st.CommandBuf)
	case sym == KeyBackSpace:
		if st.CommandBuf == "" {
			st.Command = false
		} else {
			st.CommandBuf = st.CommandBuf[:len(st.CommandBuf)-1]
		}
	case sym >= 0x20 && sym <= 0x7e: // printable ASCII keysyms equal their character
		st.CommandBuf += string(rune(sym))
	}
}

// execute runs an ex-style command line:
//
//	:w [file]       save the selection (or whole capture)
//	:wq [file], :x  save, then quit if saving succeeded
//	:q, :q!         quit
func (a *App) execute(line string) {
	cmd, arg := parseCommand(line)
	switch cmd {
	case "":
	case "q", "q!":
		a.quit = true
	case "w", "wq", "x":
		path, err := a.save(arg)
		if err != nil {
			a.State.Message = "Error: " + err.Error()
			return
		}
		a.State.Message = "Saved " + path
		a.quit = cmd != "w"
	default:
		a.State.Message = fmt.Sprintf("Not an editor command: %s", cmd)
	}
}

// parseCommand splits a command line into its command word and the rest,
// so file names may contain spaces.
func parseCommand(line string) (cmd, arg string) {
	cmd, arg, _ = strings.Cut(strings.TrimSpace(line), " ")
	return cmd, strings.TrimSpace(arg)
}
