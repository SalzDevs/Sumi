package config

import (
	"strconv"
	"strings"

	"sumi/editor"
	"sumi/registry"
)

func executeCommandLine(e *editor.Editor, r *registry.CommandRegistry) {
	input := strings.TrimSpace(e.CommandLine)
	if input == "" {
		e.SetMode(editor.ModeNormal)
		return
	}

	parts := strings.Fields(input)
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	if err := r.Execute(e, cmdName, args); err != nil {
		e.ShowError(err.Error())
	}

	e.CommandLine = ""
	e.SetMode(editor.ModeNormal)
}

// RegisterBuiltinCommands registers the minimal set of commands that must
// live in Go because they need access to internal infrastructure.
// All other commands are registered in Lua (default.lua).
func RegisterBuiltinCommands(r *registry.CommandRegistry) {
	// The command-line dispatcher — parses ":w foo" and calls the registry.
	// This needs access to the registry itself, so it must be Go.
	r.Register("execute_command", "Execute command line", 0, 0,
		func(e *editor.Editor, args []string) error {
			executeCommandLine(e, r)
			return nil
		})

	// goto_position is used by the mouse handler in main.go.
	// Kept in Go for simplicity (0-based arg parsing).
	r.Register("goto_position", "Move cursor to position", 2, 2,
		func(e *editor.Editor, args []string) error {
			line, _ := strconv.Atoi(args[0])
			col, _ := strconv.Atoi(args[1])
			if line < 0 {
				line = 0
			}
			if line >= len(e.Buffer.Lines) {
				line = len(e.Buffer.Lines) - 1
			}
			if line < 0 {
				line = 0
			}
			if col < 0 {
				col = 0
			}
			if col > e.LineLen(line) {
				col = e.LineLen(line)
			}
			e.Cursor.Line = line
			e.Cursor.Col = col
			return nil
		})
}
