package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"sumi/editor"
	"sumi/registry"

	raylib "github.com/gen2brain/raylib-go/raylib"
)

func executeCommandLine(e *editor.Editor, r *registry.CommandRegistry) {
	input := strings.TrimSpace(e.CommandLine)
	if input == "" {
		e.Mode = editor.ModeNormal
		return
	}

	parts := strings.Fields(input)
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	if err := r.Execute(e, cmdName, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	e.CommandLine = ""
	e.Mode = editor.ModeNormal
}

func RegisterBuiltinCommands(r *registry.CommandRegistry) {
	// File commands
	r.Register("w", "Save the current file", 0, 0,
		func(e *editor.Editor, args []string) error {
			return e.SaveFile()
		})

	r.Register("q", "Quit the editor", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.ShouldQuit = true
			return nil
		})

	r.Register("wq", "Save and quit", 0, 0,
		func(e *editor.Editor, args []string) error {
			if err := e.SaveFile(); err != nil {
				return err
			}
			if !e.Buffer.Modified {
				e.ShouldQuit = true
			}
			return nil
		})

	r.Register("e", "Open a file", 1, 1,
		func(e *editor.Editor, args []string) error {
			return e.LoadFile(args[0])
		})

	// Movement commands
	r.Register("move_left", "Move cursor left", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.MoveLeft()
			return nil
		})

	r.Register("move_right", "Move cursor right", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.MoveRight()
			return nil
		})

	r.Register("move_up", "Move cursor up", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.MoveUp()
			return nil
		})

	r.Register("move_down", "Move cursor down", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.MoveDown()
			return nil
		})

	// Edit commands
	r.Register("backspace", "Delete character before cursor", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.Backspace()
			return nil
		})

	r.Register("insert_newline", "Insert newline", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.InsertNewline()
			return nil
		})

	// Mode commands
	r.Register("enter_command_mode", "Enter command mode", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.Mode = editor.ModeCommand
			return nil
		})

	r.Register("cancel_command", "Cancel command mode", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.CommandLine = ""
			e.Mode = editor.ModeNormal
			return nil
		})

	r.Register("execute_command", "Execute command line", 0, 0,
		func(e *editor.Editor, args []string) error {
			executeCommandLine(e, r)
			return nil
		})

	r.Register("command_backspace", "Delete last command character", 0, 0,
		func(e *editor.Editor, args []string) error {
			if len(e.CommandLine) > 0 {
				runes := []rune(e.CommandLine)
				e.CommandLine = string(runes[:len(runes)-1])
			}
			return nil
		})

	// Undo
	r.Register("undo", "Undo last change", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.Undo()
			return nil
		})

	// Visual mode commands
	r.Register("enter_visual_mode", "Enter visual mode", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.Mode = editor.ModeVisual
			e.SetVisualAnchor()
			return nil
		})

	r.Register("cancel_visual", "Cancel visual mode", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.ClearVisual()
			return nil
		})

	r.Register("visual_delete", "Delete visual selection", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.DeleteSelection()
			return nil
		})

	r.Register("yank", "Copy selection to clipboard", 0, 0,
		func(e *editor.Editor, args []string) error {
			return e.Yank()
		})

	r.Register("paste", "Paste clipboard at cursor", 0, 0,
		func(e *editor.Editor, args []string) error {
			e.Paste()
			return nil
		})

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

func RegisterBuiltinKeymaps(k *registry.KeymapRegistry) {
	// Normal mode
	k.Register("normal", raylib.KeyLeft, "move_left")
	k.Register("normal", raylib.KeyRight, "move_right")
	k.Register("normal", raylib.KeyUp, "move_up")
	k.Register("normal", raylib.KeyDown, "move_down")
	k.Register("normal", raylib.KeyBackspace, "backspace")
	k.Register("normal", raylib.KeyEnter, "insert_newline")

	// Command mode
	k.Register("command", raylib.KeyEscape, "cancel_command")
	k.Register("command", raylib.KeyBackspace, "command_backspace")
	k.Register("command", raylib.KeyEnter, "execute_command")

	// Visual mode
	k.Register("visual", raylib.KeyEscape, "cancel_visual")
	k.Register("visual", raylib.KeyLeft, "move_left")
	k.Register("visual", raylib.KeyRight, "move_right")
	k.Register("visual", raylib.KeyUp, "move_up")
	k.Register("visual", raylib.KeyDown, "move_down")
	k.Register("visual", raylib.KeyBackspace, "backspace")

	// Ctrl+U for undo is handled in main.go alongside Ctrl+S
}
