package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	raylib "github.com/gen2brain/raylib-go/raylib"
)

const (
	fps          = 60
	screenWidth  = 800
	screenHeight = 400
	gutterWidth  = 20
	fontSize     = 20
	fontSpacing  = 1
)

const (
	modeNormal = iota
	modeCommand
)

// Command is a registered editor command.
type Command struct {
	Name        string
	Description string
	Handler     func(e *Editor, args []string) error
	MinArgs     int // minimum required arguments
	MaxArgs     int // maximum allowed arguments; -1 means unlimited
}

// CommandRegistry holds all available commands.
type CommandRegistry struct {
	commands map[string]*Command
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Command),
	}
}

// Register adds a command to the registry.
func (r *CommandRegistry) Register(name, description string, minArgs, maxArgs int, handler func(*Editor, []string) error) {
	r.commands[name] = &Command{
		Name:        name,
		Description: description,
		Handler:     handler,
		MinArgs:     minArgs,
		MaxArgs:     maxArgs,
	}
}

// Execute looks up a command by name and runs it with the given arguments.
func (r *CommandRegistry) Execute(e *Editor, name string, args []string) error {
	cmd, ok := r.commands[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}

	argCount := len(args)
	if argCount < cmd.MinArgs {
		return fmt.Errorf("%s: requires at least %d argument(s)", name, cmd.MinArgs)
	}
	if cmd.MaxArgs >= 0 && argCount > cmd.MaxArgs {
		return fmt.Errorf("%s: takes at most %d argument(s)", name, cmd.MaxArgs)
	}

	return cmd.Handler(e, args)
}

// Names returns all registered command names, sorted.
func (r *CommandRegistry) Names() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// KeymapRegistry maps (mode, key) → command name.
// V1 is single-key only. Multi-key sequences (e.g. "gg") are future work.
type KeymapRegistry struct {
	bindings map[string]map[int32]string // mode → key → command
}

func NewKeymapRegistry() *KeymapRegistry {
	return &KeymapRegistry{
		bindings: make(map[string]map[int32]string),
	}
}

// Register binds a key to a command in a given mode.
func (k *KeymapRegistry) Register(mode string, key int32, command string) {
	if k.bindings[mode] == nil {
		k.bindings[mode] = make(map[int32]string)
	}
	k.bindings[mode][key] = command
}

// Resolve looks up the command for a key in the given mode.
func (k *KeymapRegistry) Resolve(mode string, key int32) (string, bool) {
	cmds, ok := k.bindings[mode]
	if !ok {
		return "", false
	}
	cmd, ok := cmds[key]
	return cmd, ok
}

type Buffer struct {
	Lines    []string
	FilePath string
	Modified bool
}

type Cursor struct {
	Line       int
	Col        int
	DesiredCol int // -1 means unset
}

type Editor struct {
	Buffer      *Buffer
	Cursor      Cursor
	Mode        int
	CommandLine string
	ShouldQuit  bool
	Registry    *CommandRegistry
	Keymap      *KeymapRegistry
}

func NewEditor() *Editor {
	return &Editor{
		Buffer: &Buffer{
			Lines:    []string{""},
			FilePath: "./test.txt",
			Modified: false,
		},
		Cursor:      Cursor{DesiredCol: -1},
		Mode:        modeNormal,
		CommandLine: "",
		ShouldQuit:  false,
		Registry:    NewCommandRegistry(),
		Keymap:      NewKeymapRegistry(),
	}
}

func (e *Editor) modeName() string {
	switch e.Mode {
	case modeNormal:
		return "normal"
	case modeCommand:
		return "command"
	default:
		return "normal"
	}
}

func (e *Editor) resetDesired() {
	e.Cursor.DesiredCol = -1
}

func (e *Editor) lineLen(line int) int {
	return len([]rune(e.Buffer.Lines[line]))
}

func (e *Editor) moveLeft() {
	if e.Cursor.Col > 0 {
		e.Cursor.Col--
	} else if e.Cursor.Line > 0 {
		e.Cursor.Line--
		e.Cursor.Col = e.lineLen(e.Cursor.Line)
	}
	e.resetDesired()
}

func (e *Editor) moveRight() {
	if e.Cursor.Col < e.lineLen(e.Cursor.Line) {
		e.Cursor.Col++
	} else if e.Cursor.Line+1 < len(e.Buffer.Lines) {
		e.Cursor.Line++
		e.Cursor.Col = 0
	}
	e.resetDesired()
}

func (e *Editor) moveUp() {
	if e.Cursor.Line == 0 {
		return
	}
	if e.Cursor.DesiredCol < 0 {
		e.Cursor.DesiredCol = e.Cursor.Col
	}
	e.Cursor.Line--
	target := e.lineLen(e.Cursor.Line)
	if e.Cursor.DesiredCol < target {
		e.Cursor.Col = e.Cursor.DesiredCol
	} else {
		e.Cursor.Col = target
	}
}

func (e *Editor) moveDown() {
	if e.Cursor.Line+1 >= len(e.Buffer.Lines) {
		return
	}
	if e.Cursor.DesiredCol < 0 {
		e.Cursor.DesiredCol = e.Cursor.Col
	}
	e.Cursor.Line++
	target := e.lineLen(e.Cursor.Line)
	if e.Cursor.DesiredCol < target {
		e.Cursor.Col = e.Cursor.DesiredCol
	} else {
		e.Cursor.Col = target
	}
}

func (e *Editor) insertChar(ch rune) {
	line := []rune(e.Buffer.Lines[e.Cursor.Line])
	before := string(line[:e.Cursor.Col])
	after := string(line[e.Cursor.Col:])
	e.Buffer.Lines[e.Cursor.Line] = before + string(ch) + after
	e.Cursor.Col++
	e.Buffer.Modified = true
	e.resetDesired()
}

func (e *Editor) insertNewline() {
	line := []rune(e.Buffer.Lines[e.Cursor.Line])
	before := string(line[:e.Cursor.Col])
	after := string(line[e.Cursor.Col:])
	e.Buffer.Lines[e.Cursor.Line] = before

	e.Buffer.Lines = append(e.Buffer.Lines[:e.Cursor.Line+1], append([]string{after}, e.Buffer.Lines[e.Cursor.Line+1:]...)...)
	e.Cursor.Line++
	e.Cursor.Col = 0
	e.Buffer.Modified = true
	e.resetDesired()
}

func (e *Editor) backspace() {
	if e.Cursor.Col > 0 {
		line := []rune(e.Buffer.Lines[e.Cursor.Line])
		e.Buffer.Lines[e.Cursor.Line] = string(line[:e.Cursor.Col-1]) + string(line[e.Cursor.Col:])
		e.Cursor.Col--
		e.Buffer.Modified = true
		e.resetDesired()
		return
	}
	if e.Cursor.Line == 0 {
		return
	}
	prevLen := e.lineLen(e.Cursor.Line - 1)
	e.Buffer.Lines[e.Cursor.Line-1] += e.Buffer.Lines[e.Cursor.Line]
	e.Buffer.Lines = append(e.Buffer.Lines[:e.Cursor.Line], e.Buffer.Lines[e.Cursor.Line+1:]...)
	e.Cursor.Line--
	e.Cursor.Col = prevLen
	e.Buffer.Modified = true
	e.resetDesired()
}

func (e *Editor) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "\r") {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}

	e.Buffer.FilePath = path
	e.Buffer.Lines = lines
	e.Buffer.Modified = false
	e.Cursor.Line = 0
	e.Cursor.Col = 0
	e.Cursor.DesiredCol = -1
	return nil
}

func (e *Editor) saveFile() error {
	f, err := os.Create(e.Buffer.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, line := range e.Buffer.Lines {
		fmt.Fprintln(f, line)
	}
	e.Buffer.Modified = false
	return nil
}

func (e *Editor) executeCommandLine() {
	input := strings.TrimSpace(e.CommandLine)
	if input == "" {
		e.Mode = modeNormal
		return
	}

	parts := strings.Fields(input)
	cmdName := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	if err := e.Registry.Execute(e, cmdName, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	e.CommandLine = ""
	e.Mode = modeNormal
}

func registerBuiltinCommands(e *Editor) {
	r := e.Registry

	// File commands
	r.Register("w", "Save the current file", 0, 0,
		func(e *Editor, args []string) error {
			return e.saveFile()
		})

	r.Register("q", "Quit the editor", 0, 0,
		func(e *Editor, args []string) error {
			e.ShouldQuit = true
			return nil
		})

	r.Register("wq", "Save and quit", 0, 0,
		func(e *Editor, args []string) error {
			if err := e.saveFile(); err != nil {
				return err
			}
			if !e.Buffer.Modified {
				e.ShouldQuit = true
			}
			return nil
		})

	r.Register("e", "Open a file", 1, 1,
		func(e *Editor, args []string) error {
			return e.loadFile(args[0])
		})

	// Movement commands
	r.Register("move_left", "Move cursor left", 0, 0,
		func(e *Editor, args []string) error {
			e.moveLeft()
			return nil
		})

	r.Register("move_right", "Move cursor right", 0, 0,
		func(e *Editor, args []string) error {
			e.moveRight()
			return nil
		})

	r.Register("move_up", "Move cursor up", 0, 0,
		func(e *Editor, args []string) error {
			e.moveUp()
			return nil
		})

	r.Register("move_down", "Move cursor down", 0, 0,
		func(e *Editor, args []string) error {
			e.moveDown()
			return nil
		})

	// Edit commands
	r.Register("backspace", "Delete character before cursor", 0, 0,
		func(e *Editor, args []string) error {
			e.backspace()
			return nil
		})

	r.Register("insert_newline", "Insert newline", 0, 0,
		func(e *Editor, args []string) error {
			e.insertNewline()
			return nil
		})

	// Mode commands
	r.Register("enter_command_mode", "Enter command mode", 0, 0,
		func(e *Editor, args []string) error {
			e.Mode = modeCommand
			return nil
		})

	r.Register("cancel_command", "Cancel command mode", 0, 0,
		func(e *Editor, args []string) error {
			e.CommandLine = ""
			e.Mode = modeNormal
			return nil
		})

	r.Register("execute_command", "Execute command line", 0, 0,
		func(e *Editor, args []string) error {
			e.executeCommandLine()
			return nil
		})

	r.Register("command_backspace", "Delete last command character", 0, 0,
		func(e *Editor, args []string) error {
			if len(e.CommandLine) > 0 {
				runes := []rune(e.CommandLine)
				e.CommandLine = string(runes[:len(runes)-1])
			}
			return nil
		})
}

func registerBuiltinKeymaps(e *Editor) {
	k := e.Keymap

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
}

func handleInput(e *Editor) {
	mode := e.modeName()

	// --- special keys ---
	key := raylib.GetKeyPressed()
	for key != 0 {
		if cmd, ok := e.Keymap.Resolve(mode, key); ok {
			_ = e.Registry.Execute(e, cmd, nil)
		}
		// TODO: chords (Ctrl+S) should eventually live in the keymap too
		if (raylib.IsKeyDown(raylib.KeyLeftControl) || raylib.IsKeyDown(raylib.KeyRightControl)) && key == raylib.KeyS {
			_ = e.Registry.Execute(e, "w", nil)
		}
		key = raylib.GetKeyPressed()
	}

	// --- character input ---
	ch := raylib.GetCharPressed()
	for ch != 0 {
		if e.Mode == modeCommand {
			e.CommandLine += string(rune(ch))
		} else if ch == ':' {
			_ = e.Registry.Execute(e, "enter_command_mode", nil)
		} else if ch >= 32 && ch < 127 {
			e.insertChar(rune(ch))
		}
		ch = raylib.GetCharPressed()
	}
}

func renderEditor(e *Editor, font raylib.Font) {
	raylib.BeginDrawing()
	raylib.ClearBackground(raylib.RayWhite)

	penY := float32(0)
	cursorX := float32(gutterWidth)
	cursorY := float32(0)

	for lineIdx, line := range e.Buffer.Lines {
		penX := float32(gutterWidth)

		numStr := fmt.Sprintf("%d", lineIdx+1)
		raylib.DrawTextEx(font, numStr, raylib.Vector2{X: 0, Y: penY}, float32(fontSize), float32(fontSpacing), raylib.Gray)

		runes := []rune(line)
		for col, r := range runes {
			if lineIdx == e.Cursor.Line && col == e.Cursor.Col {
				cursorX = penX
				cursorY = penY
			}
			chStr := string(r)
			glyphW := raylib.MeasureTextEx(font, chStr, float32(fontSize), float32(fontSpacing)).X
			raylib.DrawTextEx(font, chStr, raylib.Vector2{X: penX, Y: penY}, float32(fontSize), float32(fontSpacing), raylib.Red)
			penX += glyphW
		}

		if lineIdx == e.Cursor.Line && e.Cursor.Col == len(runes) {
			cursorX = penX
			cursorY = penY
		}

		penY += float32(fontSize)
	}

	raylib.DrawRectangle(int32(cursorX), int32(cursorY), 2, fontSize, raylib.Green)

	if e.Mode == modeCommand {
		cmdHeight := fontSize + 4
		cmdY := screenHeight - cmdHeight
		raylib.DrawRectangle(0, int32(cmdY), screenWidth, int32(cmdHeight), raylib.LightGray)
		prompt := fmt.Sprintf(":%s", e.CommandLine)
		raylib.DrawTextEx(font, prompt, raylib.Vector2{X: 4, Y: float32(cmdY) + 2}, float32(fontSize), float32(fontSpacing), raylib.Black)
	}

	raylib.EndDrawing()
}

func main() {
	editor := NewEditor()
	registerBuiltinCommands(editor)
	registerBuiltinKeymaps(editor)

	if len(os.Args) > 1 {
		_ = editor.loadFile(os.Args[1])
	} else {
		_ = editor.loadFile("./test.txt")
	}

	raylib.InitWindow(screenWidth, screenHeight, "Sumi")
	raylib.SetTargetFPS(fps)
	defer raylib.CloseWindow()

	font := raylib.LoadFontEx("assets/JetBrainsMono-Regular.ttf", fontSize, nil, 0)
	if font.Texture.ID == 0 {
		font = raylib.GetFontDefault()
	}
	defer raylib.UnloadFont(font)

	for !raylib.WindowShouldClose() && !editor.ShouldQuit {
		handleInput(editor)
		renderEditor(editor, font)
	}
}
