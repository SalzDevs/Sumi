package lua

import (
	_ "embed"
	"fmt"
	"os"

	glua "github.com/yuin/gopher-lua"
	"sumi/editor"
	"sumi/registry"
)

//go:embed default.lua
var defaultLua string

// Bridge connects the Go engine to the Lua configuration layer.
type Bridge struct {
	L       *glua.LState
	Editor  *editor.Editor
	CmdReg  *registry.CommandRegistry
	KeyReg  *registry.KeymapRegistry
	luaCmds map[string]*glua.LFunction // command name → Lua handler
}

// NewBridge creates a Lua state and exposes the editor API.
func NewBridge(ed *editor.Editor, cmdReg *registry.CommandRegistry, keyReg *registry.KeymapRegistry) *Bridge {
	L := glua.NewState()
	b := &Bridge{
		L:       L,
		Editor:  ed,
		CmdReg:  cmdReg,
		KeyReg:  keyReg,
		luaCmds: make(map[string]*glua.LFunction),
	}
	b.registerAPI()
	return b
}

func (b *Bridge) registerAPI() {
	// keymap:Register(mode, keyCode, commandName)
	km := b.L.NewTable()
	b.L.SetField(km, "Register", b.L.NewFunction(b.luaKeymapRegister))
	b.L.SetGlobal("keymap", km)

	// commands:Register(name, desc, minArgs, maxArgs, handler)
	// commands:List() → array of command names
	cmds := b.L.NewTable()
	b.L.SetField(cmds, "Register", b.L.NewFunction(b.luaCommandRegister))
	b.L.SetField(cmds, "List", b.L.NewFunction(b.luaCommandList))
	b.L.SetGlobal("commands", cmds)

	// keys constants
	keys := b.L.NewTable()
	b.L.SetField(keys, "RIGHT", glua.LNumber(262))
	b.L.SetField(keys, "LEFT", glua.LNumber(263))
	b.L.SetField(keys, "DOWN", glua.LNumber(264))
	b.L.SetField(keys, "UP", glua.LNumber(265))
	b.L.SetField(keys, "ENTER", glua.LNumber(257))
	b.L.SetField(keys, "ESCAPE", glua.LNumber(256))
	b.L.SetField(keys, "BACKSPACE", glua.LNumber(259))
	b.L.SetField(keys, "HOME", glua.LNumber(268))
	b.L.SetField(keys, "END", glua.LNumber(269))
	b.L.SetGlobal("keys", keys)

	// editor table (live mutable API)
	edTbl := b.L.NewTable()
	b.L.SetField(edTbl, "Mode", b.L.NewFunction(b.luaEditorMode))
	b.L.SetField(edTbl, "LineCount", b.L.NewFunction(b.luaEditorLineCount))
	b.L.SetField(edTbl, "LoadFile", b.L.NewFunction(b.luaEditorLoadFile))
	b.L.SetField(edTbl, "SaveFile", b.L.NewFunction(b.luaEditorSaveFile))
	b.L.SetField(edTbl, "Undo", b.L.NewFunction(b.luaEditorUndo))
	b.L.SetField(edTbl, "Quit", b.L.NewFunction(b.luaEditorQuit))
	b.L.SetField(edTbl, "EnterVisual", b.L.NewFunction(b.luaEditorEnterVisual))
	b.L.SetField(edTbl, "ClearVisual", b.L.NewFunction(b.luaEditorClearVisual))

	// editor.Buffer
	bufTbl := b.L.NewTable()
	b.L.SetField(bufTbl, "GetLine", b.L.NewFunction(b.luaBufferGetLine))
	b.L.SetField(bufTbl, "SetLine", b.L.NewFunction(b.luaBufferSetLine))
	b.L.SetField(bufTbl, "LineCount", b.L.NewFunction(b.luaBufferLineCount))
	b.L.SetField(bufTbl, "InsertChar", b.L.NewFunction(b.luaBufferInsertChar))
	b.L.SetField(bufTbl, "DeleteChar", b.L.NewFunction(b.luaBufferDeleteChar))
	b.L.SetField(edTbl, "Buffer", bufTbl)

	// editor.Cursor
	curTbl := b.L.NewTable()
	b.L.SetField(curTbl, "Line", b.L.NewFunction(b.luaCursorLine))
	b.L.SetField(curTbl, "Col", b.L.NewFunction(b.luaCursorCol))
	b.L.SetField(curTbl, "Goto", b.L.NewFunction(b.luaCursorGoto))
	b.L.SetField(curTbl, "MoveLeft", b.L.NewFunction(b.luaCursorMoveLeft))
	b.L.SetField(curTbl, "MoveRight", b.L.NewFunction(b.luaCursorMoveRight))
	b.L.SetField(curTbl, "MoveUp", b.L.NewFunction(b.luaCursorMoveUp))
	b.L.SetField(curTbl, "MoveDown", b.L.NewFunction(b.luaCursorMoveDown))
	b.L.SetField(edTbl, "Cursor", curTbl)

	b.L.SetGlobal("editor", edTbl)
}

// -------------------------------------------------------------------------
// keymap
// -------------------------------------------------------------------------

func (b *Bridge) luaKeymapRegister(L *glua.LState) int {
	_ = L.CheckAny(1)
	mode := L.CheckString(2)
	keyCode := L.CheckInt(3)
	cmdName := L.CheckString(4)
	b.KeyReg.Register(mode, int32(keyCode), cmdName)
	return 0
}

// -------------------------------------------------------------------------
// commands
// -------------------------------------------------------------------------

func (b *Bridge) luaCommandRegister(L *glua.LState) int {
	_ = L.CheckAny(1)
	name := L.CheckString(2)
	desc := L.CheckString(3)
	minArgs := L.CheckInt(4)
	maxArgs := L.CheckInt(5)
	fn := L.CheckFunction(6)

	b.luaCmds[name] = fn
	b.CmdReg.Register(name, desc, minArgs, maxArgs, func(e *editor.Editor, args []string) error {
		return b.callLuaCommand(name, e, args)
	})
	return 0
}

func (b *Bridge) callLuaCommand(name string, e *editor.Editor, args []string) error {
	fn, ok := b.luaCmds[name]
	if !ok {
		return fmt.Errorf("lua command %q not found", name)
	}

	b.L.Push(fn)
	b.pushEditorAPI(e)

	argsTbl := b.L.NewTable()
	for i, arg := range args {
		b.L.RawSetInt(argsTbl, i+1, glua.LString(arg))
	}
	b.L.Push(argsTbl)

	if err := b.L.PCall(2, 1, nil); err != nil {
		return fmt.Errorf("lua: %v", err)
	}

	ret := b.L.Get(-1)
	b.L.Pop(1)

	if ret == glua.LNil {
		return nil
	}
	if s, ok := ret.(glua.LString); ok && string(s) != "" {
		return fmt.Errorf("%s", string(s))
	}
	return nil
}

func (b *Bridge) luaCommandList(L *glua.LState) int {
	_ = L.CheckAny(1)
	names := b.CmdReg.Names()
	tbl := b.L.NewTable()
	for i, name := range names {
		b.L.RawSetInt(tbl, i+1, glua.LString(name))
	}
	b.L.Push(tbl)
	return 1
}

// -------------------------------------------------------------------------
// editor helpers
// -------------------------------------------------------------------------

func (b *Bridge) pushEditorAPI(e *editor.Editor) {
	// Build a lightweight editor proxy for the current editor instance.
	// This is used when calling Lua-registered command handlers.
	edTbl := b.L.NewTable()
	b.L.SetField(edTbl, "Mode", glua.LString(func() string {
		switch e.Mode {
		case editor.ModeNormal:
			return "normal"
		case editor.ModeCommand:
			return "command"
		case editor.ModeVisual:
			return "visual"
		default:
			return "normal"
		}
	}()))
	b.L.SetField(edTbl, "LineCount", glua.LNumber(len(e.Buffer.Lines)))

	bufTbl := b.L.NewTable()
	b.L.SetField(bufTbl, "GetLine", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		n := L.CheckInt(2) - 1 // 1-based → 0-based
		if n < 0 || n >= len(e.Buffer.Lines) {
			L.Push(glua.LString(""))
		} else {
			L.Push(glua.LString(e.Buffer.Lines[n]))
		}
		return 1
	}))
	b.L.SetField(bufTbl, "SetLine", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		n := L.CheckInt(2) - 1
		text := L.CheckString(3)
		if n >= 0 && n < len(e.Buffer.Lines) {
			e.Buffer.Lines[n] = text
			e.Buffer.Modified = true
		}
		return 0
	}))
	b.L.SetField(bufTbl, "LineCount", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LNumber(len(e.Buffer.Lines)))
		return 1
	}))
	b.L.SetField(bufTbl, "InsertChar", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		line := L.CheckInt(2) - 1
		col := L.CheckInt(3) - 1
		ch := L.CheckString(4)
		if line >= 0 && line < len(e.Buffer.Lines) && len(ch) > 0 {
			runes := []rune(e.Buffer.Lines[line])
			if col >= 0 && col <= len(runes) {
				c := runes[:col]
				c = append(c, rune(ch[0]))
				c = append(c, runes[col:]...)
				e.Buffer.Lines[line] = string(c)
				e.Buffer.Modified = true
			}
		}
		return 0
	}))
	b.L.SetField(bufTbl, "DeleteChar", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		line := L.CheckInt(2) - 1
		col := L.CheckInt(3) - 1
		if line >= 0 && line < len(e.Buffer.Lines) {
			runes := []rune(e.Buffer.Lines[line])
			if col >= 0 && col < len(runes) {
				runes = append(runes[:col], runes[col+1:]...)
				e.Buffer.Lines[line] = string(runes)
				e.Buffer.Modified = true
			}
		}
		return 0
	}))
	b.L.SetField(edTbl, "Buffer", bufTbl)

	curTbl := b.L.NewTable()
	b.L.SetField(curTbl, "Line", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LNumber(e.Cursor.Line + 1))
		return 1
	}))
	b.L.SetField(curTbl, "Col", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LNumber(e.Cursor.Col + 1))
		return 1
	}))
	b.L.SetField(curTbl, "Goto", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		line := L.CheckInt(2) - 1
		col := L.CheckInt(3) - 1
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
		if col > len([]rune(e.Buffer.Lines[line])) {
			col = len([]rune(e.Buffer.Lines[line]))
		}
		e.Cursor.Line = line
		e.Cursor.Col = col
		return 0
	}))
	b.L.SetField(curTbl, "MoveLeft", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.MoveLeft()
		return 0
	}))
	b.L.SetField(curTbl, "MoveRight", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.MoveRight()
		return 0
	}))
	b.L.SetField(curTbl, "MoveUp", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.MoveUp()
		return 0
	}))
	b.L.SetField(curTbl, "MoveDown", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.MoveDown()
		return 0
	}))
	b.L.SetField(edTbl, "Cursor", curTbl)

	b.L.Push(edTbl)
}

// -------------------------------------------------------------------------
// editor methods (for the global editor table, bound at startup)
// -------------------------------------------------------------------------

func (b *Bridge) luaEditorMode(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LString(b.Editor.ModeName()))
	return 1
}

func (b *Bridge) luaEditorLineCount(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(len(b.Editor.Buffer.Lines)))
	return 1
}

func (b *Bridge) luaEditorLoadFile(L *glua.LState) int {
	_ = L.CheckAny(1)
	path := L.CheckString(2)
	if err := b.Editor.LoadFile(path); err != nil {
		L.Push(glua.LString(err.Error()))
		return 1
	}
	return 0
}

func (b *Bridge) luaEditorSaveFile(L *glua.LState) int {
	_ = L.CheckAny(1)
	if err := b.Editor.SaveFile(); err != nil {
		L.Push(glua.LString(err.Error()))
		return 1
	}
	return 0
}

func (b *Bridge) luaEditorUndo(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.Undo()
	return 0
}

func (b *Bridge) luaEditorQuit(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.ShouldQuit = true
	return 0
}

func (b *Bridge) luaEditorEnterVisual(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.Mode = editor.ModeVisual
	b.Editor.SetVisualAnchor()
	return 0
}

func (b *Bridge) luaEditorClearVisual(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.ClearVisual()
	return 0
}

// -------------------------------------------------------------------------
// Buffer methods (for the global editor.Buffer table)
// -------------------------------------------------------------------------

func (b *Bridge) luaBufferGetLine(L *glua.LState) int {
	_ = L.CheckAny(1)
	n := L.CheckInt(2) - 1
	if n < 0 || n >= len(b.Editor.Buffer.Lines) {
		L.Push(glua.LString(""))
	} else {
		L.Push(glua.LString(b.Editor.Buffer.Lines[n]))
	}
	return 1
}

func (b *Bridge) luaBufferSetLine(L *glua.LState) int {
	_ = L.CheckAny(1)
	n := L.CheckInt(2) - 1
	text := L.CheckString(3)
	if n >= 0 && n < len(b.Editor.Buffer.Lines) {
		b.Editor.Buffer.Lines[n] = text
		b.Editor.Buffer.Modified = true
	}
	return 0
}

func (b *Bridge) luaBufferLineCount(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(len(b.Editor.Buffer.Lines)))
	return 1
}

func (b *Bridge) luaBufferInsertChar(L *glua.LState) int {
	_ = L.CheckAny(1)
	line := L.CheckInt(2) - 1
	col := L.CheckInt(3) - 1
	ch := L.CheckString(4)
	if line >= 0 && line < len(b.Editor.Buffer.Lines) && len(ch) > 0 {
		runes := []rune(b.Editor.Buffer.Lines[line])
		if col >= 0 && col <= len(runes) {
			c := runes[:col]
			c = append(c, rune(ch[0]))
			c = append(c, runes[col:]...)
			b.Editor.Buffer.Lines[line] = string(c)
			b.Editor.Buffer.Modified = true
		}
	}
	return 0
}

func (b *Bridge) luaBufferDeleteChar(L *glua.LState) int {
	_ = L.CheckAny(1)
	line := L.CheckInt(2) - 1
	col := L.CheckInt(3) - 1
	if line >= 0 && line < len(b.Editor.Buffer.Lines) {
		runes := []rune(b.Editor.Buffer.Lines[line])
		if col >= 0 && col < len(runes) {
			runes = append(runes[:col], runes[col+1:]...)
			b.Editor.Buffer.Lines[line] = string(runes)
			b.Editor.Buffer.Modified = true
		}
	}
	return 0
}

// -------------------------------------------------------------------------
// Cursor methods
// -------------------------------------------------------------------------

func (b *Bridge) luaCursorLine(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(b.Editor.Cursor.Line + 1))
	return 1
}

func (b *Bridge) luaCursorCol(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(b.Editor.Cursor.Col + 1))
	return 1
}

func (b *Bridge) luaCursorGoto(L *glua.LState) int {
	_ = L.CheckAny(1)
	line := L.CheckInt(2) - 1
	col := L.CheckInt(3) - 1
	if line < 0 {
		line = 0
	}
	if line >= len(b.Editor.Buffer.Lines) {
		line = len(b.Editor.Buffer.Lines) - 1
	}
	if line < 0 {
		line = 0
	}
	if col < 0 {
		col = 0
	}
	if col > len([]rune(b.Editor.Buffer.Lines[line])) {
		col = len([]rune(b.Editor.Buffer.Lines[line]))
	}
	b.Editor.Cursor.Line = line
	b.Editor.Cursor.Col = col
	return 0
}

func (b *Bridge) luaCursorMoveLeft(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.MoveLeft()
	return 0
}

func (b *Bridge) luaCursorMoveRight(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.MoveRight()
	return 0
}

func (b *Bridge) luaCursorMoveUp(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.MoveUp()
	return 0
}

func (b *Bridge) luaCursorMoveDown(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.MoveDown()
	return 0
}

// -------------------------------------------------------------------------
// Config loading
// -------------------------------------------------------------------------

// LoadDefaults runs the embedded default.lua configuration.
func (b *Bridge) LoadDefaults() error {
	return b.L.DoString(defaultLua)
}

// LoadUserConfig runs ~/.config/sumi/init.lua if it exists.
func (b *Bridge) LoadUserConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := home + "/.config/sumi/init.lua"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return b.L.DoFile(path)
}

// Close shuts down the Lua state.
func (b *Bridge) Close() {
	b.L.Close()
}

// FallbackKeymaps registers Go keymaps when Lua loading fails.
func FallbackKeymaps(keyReg *registry.KeymapRegistry) {
	fmt.Fprintln(os.Stderr, "Lua config failed; falling back to built-in keymaps")
	keyReg.Register("normal", 262, "move_right")
	keyReg.Register("normal", 263, "move_left")
	keyReg.Register("normal", 264, "move_down")
	keyReg.Register("normal", 265, "move_up")
	keyReg.Register("normal", 259, "backspace")
	keyReg.Register("normal", 257, "insert_newline")
	keyReg.Register("command", 256, "cancel_command")
	keyReg.Register("command", 259, "command_backspace")
	keyReg.Register("command", 257, "execute_command")
	keyReg.Register("visual", 256, "cancel_visual")
	keyReg.Register("visual", 262, "move_left")
	keyReg.Register("visual", 263, "move_right")
	keyReg.Register("visual", 264, "move_down")
	keyReg.Register("visual", 265, "move_up")
	keyReg.Register("visual", 259, "backspace")
}
