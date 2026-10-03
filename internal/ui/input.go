package ui

import (
	"fmt"
	"strings"

	"github.com/jezek/xgb/xproto"
)

const (
	xkBackSpace xproto.Keysym = 0xff08
	xkReturn    xproto.Keysym = 0xff0d
	xkEscape    xproto.Keysym = 0xff1b
	xkKPEnter   xproto.Keysym = 0xff8d
)

func (a *app) key(sym xproto.Keysym) {
	st := a.st
	if st.Command {
		a.commandKey(sym)
		return
	}

	switch sym {
	case xkEscape:
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

func (a *app) commandKey(sym xproto.Keysym) {
	st := a.st
	switch {
	case sym == xkEscape:
		st.Command = false
	case sym == xkReturn || sym == xkKPEnter:
		st.Command = false
		a.execute(st.CommandBuf)
	case sym == xkBackSpace:
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
func (a *app) execute(line string) {
	cmd, arg := parseCommand(line)
	switch cmd {
	case "":
	case "q", "q!":
		a.quit = true
	case "w", "wq", "x":
		path, err := a.save(arg)
		if err != nil {
			a.st.Message = "Error: " + err.Error()
			return
		}
		a.st.Message = "Saved " + path
		a.quit = cmd != "w"
	default:
		a.st.Message = fmt.Sprintf("Not an editor command: %s", cmd)
	}
}

// parseCommand splits a command line into its command word and the rest,
// so file names may contain spaces.
func parseCommand(line string) (cmd, arg string) {
	cmd, arg, _ = strings.Cut(strings.TrimSpace(line), " ")
	return cmd, strings.TrimSpace(arg)
}
