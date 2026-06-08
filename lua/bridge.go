package lua

import (
	_ "embed"
	"fmt"
	"os"
	"strconv"
	"strings"

	glua "github.com/yuin/gopher-lua"
	raylib "github.com/gen2brain/raylib-go/raylib"
	"sumi/editor"
	"sumi/registry"
	"sumi/render"
)

//go:embed default.lua
var defaultLua string

// Bridge connects the Go engine to the Lua configuration layer.
type Bridge struct {
	L            *glua.LState
	Editor       *editor.Editor
	CmdReg       *registry.CommandRegistry
	KeyReg       *registry.KeymapRegistry
	luaCmds      map[string]*glua.LFunction // command name → Lua handler
	renderHookFn *glua.LFunction
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
	b.L.SetField(edTbl, "SetMode", b.L.NewFunction(b.luaEditorSetMode))
	b.L.SetField(edTbl, "LineCount", b.L.NewFunction(b.luaEditorLineCount))
	b.L.SetField(edTbl, "Modified", b.L.NewFunction(b.luaEditorModified))
	b.L.SetField(edTbl, "CommandLine", b.L.NewFunction(b.luaEditorCommandLine))
	b.L.SetField(edTbl, "SetCommandLine", b.L.NewFunction(b.luaEditorSetCommandLine))
	b.L.SetField(edTbl, "CommandLineBackspace", b.L.NewFunction(b.luaEditorCommandLineBackspace))
	b.L.SetField(edTbl, "LoadFile", b.L.NewFunction(b.luaEditorLoadFile))
	b.L.SetField(edTbl, "SaveFile", b.L.NewFunction(b.luaEditorSaveFile))
	b.L.SetField(edTbl, "Undo", b.L.NewFunction(b.luaEditorUndo))
	b.L.SetField(edTbl, "Quit", b.L.NewFunction(b.luaEditorQuit))
	b.L.SetField(edTbl, "EnterVisual", b.L.NewFunction(b.luaEditorEnterVisual))
	b.L.SetField(edTbl, "ClearVisual", b.L.NewFunction(b.luaEditorClearVisual))
	b.L.SetField(edTbl, "SetVisualAnchor", b.L.NewFunction(b.luaEditorSetVisualAnchor))
	b.L.SetField(edTbl, "SelectWordAt", b.L.NewFunction(b.luaEditorSelectWordAt))
	b.L.SetField(edTbl, "SelectLineAt", b.L.NewFunction(b.luaEditorSelectLineAt))

	// editor.Buffer
	bufTbl := b.L.NewTable()
	b.L.SetField(bufTbl, "GetLine", b.L.NewFunction(b.luaBufferGetLine))
	b.L.SetField(bufTbl, "SetLine", b.L.NewFunction(b.luaBufferSetLine))
	b.L.SetField(bufTbl, "LineCount", b.L.NewFunction(b.luaBufferLineCount))
	b.L.SetField(bufTbl, "InsertChar", b.L.NewFunction(b.luaBufferInsertChar))
	b.L.SetField(bufTbl, "DeleteChar", b.L.NewFunction(b.luaBufferDeleteChar))
	b.L.SetField(edTbl, "Buffer", bufTbl)

	// editor editing
	b.L.SetField(edTbl, "Backspace", b.L.NewFunction(b.luaEditorBackspace))
	b.L.SetField(edTbl, "InsertNewline", b.L.NewFunction(b.luaEditorInsertNewline))
	b.L.SetField(edTbl, "InsertChar", b.L.NewFunction(b.luaEditorInsertChar))
	b.L.SetField(edTbl, "Yank", b.L.NewFunction(b.luaEditorYank))
	b.L.SetField(edTbl, "Paste", b.L.NewFunction(b.luaEditorPaste))
	b.L.SetField(edTbl, "DeleteSelection", b.L.NewFunction(b.luaEditorDeleteSelection))

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

	b.registerRenderAPI()
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
	b.pushEditorProxy(e)

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
// pushEditorProxy — comprehensive proxy passed to Lua command handlers
// -------------------------------------------------------------------------

func (b *Bridge) pushEditorProxy(e *editor.Editor) {
	edTbl := b.L.NewTable()

	b.L.SetField(edTbl, "Mode", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LString(func() string {
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
		return 1
	}))

	b.L.SetField(edTbl, "SetMode", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		mode := L.CheckString(2)
		switch mode {
		case "normal":
			e.Mode = editor.ModeNormal
		case "command":
			e.Mode = editor.ModeCommand
		case "visual":
			e.Mode = editor.ModeVisual
		}
		return 0
	}))

	b.L.SetField(edTbl, "LineCount", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LNumber(len(e.Buffer.Lines)))
		return 1
	}))

	b.L.SetField(edTbl, "Modified", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LBool(e.Buffer.Modified))
		return 1
	}))

	b.L.SetField(edTbl, "CommandLine", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LString(e.CommandLine))
		return 1
	}))

	b.L.SetField(edTbl, "SetCommandLine", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.CommandLine = L.CheckString(2)
		return 0
	}))

	b.L.SetField(edTbl, "CommandLineBackspace", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		if len(e.CommandLine) > 0 {
			runes := []rune(e.CommandLine)
			e.CommandLine = string(runes[:len(runes)-1])
		}
		return 0
	}))

	b.L.SetField(edTbl, "LoadFile", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		path := L.CheckString(2)
		if err := e.LoadFile(path); err != nil {
			L.Push(glua.LString(err.Error()))
			return 1
		}
		return 0
	}))

	b.L.SetField(edTbl, "SaveFile", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		if err := e.SaveFile(); err != nil {
			L.Push(glua.LString(err.Error()))
			return 1
		}
		return 0
	}))

	b.L.SetField(edTbl, "Undo", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.Undo()
		return 0
	}))

	b.L.SetField(edTbl, "Quit", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.ShouldQuit = true
		return 0
	}))

	b.L.SetField(edTbl, "EnterVisual", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.Mode = editor.ModeVisual
		e.SetVisualAnchor()
		return 0
	}))

	b.L.SetField(edTbl, "ClearVisual", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.ClearVisual()
		return 0
	}))

	b.L.SetField(edTbl, "SetVisualAnchor", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.SetVisualAnchor()
		return 0
	}))

	b.L.SetField(edTbl, "SelectWordAt", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		line := L.CheckInt(2) - 1
		col := L.CheckInt(3) - 1
		e.SelectWordAt(line, col)
		return 0
	}))

	b.L.SetField(edTbl, "SelectLineAt", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		line := L.CheckInt(2) - 1
		e.SelectLineAt(line)
		return 0
	}))

	// Buffer
	bufTbl := b.L.NewTable()
	b.L.SetField(bufTbl, "GetLine", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		n := L.CheckInt(2) - 1
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

	// Editing methods
	b.L.SetField(edTbl, "Backspace", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.Backspace()
		return 0
	}))
	b.L.SetField(edTbl, "InsertNewline", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.InsertNewline()
		return 0
	}))
	b.L.SetField(edTbl, "InsertChar", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		ch := L.CheckString(2)
		if len(ch) > 0 {
			e.InsertChar(rune(ch[0]))
		}
		return 0
	}))
	b.L.SetField(edTbl, "Yank", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		if err := e.Yank(); err != nil {
			L.Push(glua.LString(err.Error()))
			return 1
		}
		return 0
	}))
	b.L.SetField(edTbl, "Paste", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.Paste()
		return 0
	}))
	b.L.SetField(edTbl, "DeleteSelection", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.DeleteSelection()
		return 0
	}))

	// Cursor
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
// editor methods (global table)
// -------------------------------------------------------------------------

func (b *Bridge) luaEditorMode(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LString(b.Editor.ModeName()))
	return 1
}

func (b *Bridge) luaEditorSetMode(L *glua.LState) int {
	_ = L.CheckAny(1)
	mode := L.CheckString(2)
	switch mode {
	case "normal":
		b.Editor.Mode = editor.ModeNormal
	case "command":
		b.Editor.Mode = editor.ModeCommand
	case "visual":
		b.Editor.Mode = editor.ModeVisual
	}
	return 0
}

func (b *Bridge) luaEditorLineCount(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(len(b.Editor.Buffer.Lines)))
	return 1
}

func (b *Bridge) luaEditorModified(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LBool(b.Editor.Buffer.Modified))
	return 1
}

func (b *Bridge) luaEditorCommandLine(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LString(b.Editor.CommandLine))
	return 1
}

func (b *Bridge) luaEditorSetCommandLine(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.CommandLine = L.CheckString(2)
	return 0
}

func (b *Bridge) luaEditorCommandLineBackspace(L *glua.LState) int {
	_ = L.CheckAny(1)
	if len(b.Editor.CommandLine) > 0 {
		runes := []rune(b.Editor.CommandLine)
		b.Editor.CommandLine = string(runes[:len(runes)-1])
	}
	return 0
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

func (b *Bridge) luaEditorSetVisualAnchor(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.SetVisualAnchor()
	return 0
}

func (b *Bridge) luaEditorSelectWordAt(L *glua.LState) int {
	_ = L.CheckAny(1)
	line := L.CheckInt(2) - 1
	col := L.CheckInt(3) - 1
	b.Editor.SelectWordAt(line, col)
	return 0
}

func (b *Bridge) luaEditorSelectLineAt(L *glua.LState) int {
	_ = L.CheckAny(1)
	line := L.CheckInt(2) - 1
	b.Editor.SelectLineAt(line)
	return 0
}

func (b *Bridge) luaEditorBackspace(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.Backspace()
	return 0
}

func (b *Bridge) luaEditorInsertNewline(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.InsertNewline()
	return 0
}

func (b *Bridge) luaEditorInsertChar(L *glua.LState) int {
	_ = L.CheckAny(1)
	ch := L.CheckString(2)
	if len(ch) > 0 {
		b.Editor.InsertChar(rune(ch[0]))
	}
	return 0
}

func (b *Bridge) luaEditorYank(L *glua.LState) int {
	_ = L.CheckAny(1)
	if err := b.Editor.Yank(); err != nil {
		L.Push(glua.LString(err.Error()))
		return 1
	}
	return 0
}

func (b *Bridge) luaEditorPaste(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.Paste()
	return 0
}

func (b *Bridge) luaEditorDeleteSelection(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.DeleteSelection()
	return 0
}

// -------------------------------------------------------------------------
// Buffer methods (global table)
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
// Render API
// -------------------------------------------------------------------------

func (b *Bridge) registerRenderAPI() {
	renderTbl := b.L.NewTable()
	b.L.SetField(renderTbl, "SetCallback", b.L.NewFunction(b.luaRenderSetCallback))
	b.L.SetField(renderTbl, "DrawRectangle", b.L.NewFunction(b.luaRenderDrawRectangle))
	b.L.SetField(renderTbl, "DrawText", b.L.NewFunction(b.luaRenderDrawText))
	b.L.SetField(renderTbl, "DrawLine", b.L.NewFunction(b.luaRenderDrawLine))
	b.L.SetField(renderTbl, "MeasureText", b.L.NewFunction(b.luaRenderMeasureText))
	b.L.SetField(renderTbl, "ScreenWidth", b.L.NewFunction(b.luaRenderScreenWidth))
	b.L.SetField(renderTbl, "ScreenHeight", b.L.NewFunction(b.luaRenderScreenHeight))
	b.L.SetField(renderTbl, "Color", b.L.NewFunction(b.luaRenderColor))
	b.L.SetGlobal("render", renderTbl)
}

func (b *Bridge) luaRenderSetCallback(L *glua.LState) int {
	_ = L.CheckAny(1)
	fn := L.CheckFunction(2)
	b.renderHookFn = fn
	b.Editor.RenderHook = func() {
		if b.renderHookFn == nil {
			return
		}
		b.L.Push(b.renderHookFn)
		if err := b.L.PCall(0, 0, nil); err != nil {
			fmt.Fprintf(os.Stderr, "render hook error: %v\n", err)
			b.renderHookFn = nil
			b.Editor.RenderHook = nil
		}
	}
	return 0
}

func luaColorToRaylib(v glua.LValue) raylib.Color {
	if n, ok := v.(glua.LNumber); ok {
		c := uint32(n)
		return raylib.NewColor(
			uint8((c>>24)&0xFF),
			uint8((c>>16)&0xFF),
			uint8((c>>8)&0xFF),
			uint8(c&0xFF),
		)
	}
	s := strings.TrimPrefix(string(v.(glua.LString)), "#")
	if len(s) == 6 {
		if v, err := strconv.ParseUint(s, 16, 32); err == nil {
			return raylib.NewColor(
				uint8((v>>16)&0xFF),
				uint8((v>>8)&0xFF),
				uint8(v&0xFF),
				255,
			)
		}
	}
	if len(s) == 8 {
		if v, err := strconv.ParseUint(s, 16, 32); err == nil {
			return raylib.NewColor(
				uint8((v>>24)&0xFF),
				uint8((v>>16)&0xFF),
				uint8((v>>8)&0xFF),
				uint8(v&0xFF),
			)
		}
	}
	return raylib.NewColor(255, 255, 255, 255)
}

func (b *Bridge) luaRenderDrawRectangle(L *glua.LState) int {
	if !render.IsDrawing() {
		return 0
	}
	_ = L.CheckAny(1)
	x := float32(L.CheckNumber(2))
	y := float32(L.CheckNumber(3))
	w := float32(L.CheckNumber(4))
	h := float32(L.CheckNumber(5))
	color := luaColorToRaylib(L.Get(6))
	raylib.DrawRectangle(int32(x), int32(y), int32(w), int32(h), color)
	return 0
}

func (b *Bridge) luaRenderDrawText(L *glua.LState) int {
	if !render.IsDrawing() {
		return 0
	}
	_ = L.CheckAny(1)
	text := L.CheckString(2)
	x := float32(L.CheckNumber(3))
	y := float32(L.CheckNumber(4))
	size := float32(L.CheckNumber(5))
	color := luaColorToRaylib(L.Get(6))
	font := render.ActiveFont()
	if font.Texture.ID == 0 {
		font = raylib.GetFontDefault()
	}
	raylib.DrawTextEx(font, text, raylib.Vector2{X: x, Y: y}, size, float32(render.FontSpacing), color)
	return 0
}

func (b *Bridge) luaRenderDrawLine(L *glua.LState) int {
	if !render.IsDrawing() {
		return 0
	}
	_ = L.CheckAny(1)
	x1 := float32(L.CheckNumber(2))
	y1 := float32(L.CheckNumber(3))
	x2 := float32(L.CheckNumber(4))
	y2 := float32(L.CheckNumber(5))
	color := luaColorToRaylib(L.Get(6))
	raylib.DrawLine(int32(x1), int32(y1), int32(x2), int32(y2), color)
	return 0
}

func (b *Bridge) luaRenderMeasureText(L *glua.LState) int {
	_ = L.CheckAny(1)
	text := L.CheckString(2)
	size := float32(L.CheckNumber(3))
	font := render.ActiveFont()
	if font.Texture.ID == 0 {
		font = raylib.GetFontDefault()
	}
	w := raylib.MeasureTextEx(font, text, size, float32(render.FontSpacing)).X
	L.Push(glua.LNumber(w))
	return 1
}

func (b *Bridge) luaRenderScreenWidth(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(render.ScreenWidth()))
	return 1
}

func (b *Bridge) luaRenderScreenHeight(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(render.ScreenHeight()))
	return 1
}

func (b *Bridge) luaRenderColor(L *glua.LState) int {
	_ = L.CheckAny(1)
	r := uint8(L.CheckNumber(2))
	g := uint8(L.CheckNumber(3))
	bl := uint8(L.CheckNumber(4))
	a := uint8(255)
	if L.GetTop() >= 5 {
		a = uint8(L.CheckNumber(5))
	}
	packed := uint32(r)<<24 | uint32(g)<<16 | uint32(bl)<<8 | uint32(a)
	L.Push(glua.LNumber(packed))
	return 1
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

// FallbackKeymaps bootstraps a minimal editor when Lua loading fails.
// It registers both commands and keymaps in Go so the editor is usable
// even if the Lua runtime is broken.
func FallbackKeymaps(cmdReg *registry.CommandRegistry, keyReg *registry.KeymapRegistry) {
	fmt.Fprintln(os.Stderr, "Lua config failed; falling back to built-in defaults")

	// Movement commands
	cmdReg.Register("move_left", "Move cursor left", 0, 0, func(e *editor.Editor, args []string) error { e.MoveLeft(); return nil })
	cmdReg.Register("move_right", "Move cursor right", 0, 0, func(e *editor.Editor, args []string) error { e.MoveRight(); return nil })
	cmdReg.Register("move_up", "Move cursor up", 0, 0, func(e *editor.Editor, args []string) error { e.MoveUp(); return nil })
	cmdReg.Register("move_down", "Move cursor down", 0, 0, func(e *editor.Editor, args []string) error { e.MoveDown(); return nil })

	// Edit commands
	cmdReg.Register("backspace", "Delete character before cursor", 0, 0, func(e *editor.Editor, args []string) error { e.Backspace(); return nil })
	cmdReg.Register("insert_newline", "Insert newline", 0, 0, func(e *editor.Editor, args []string) error { e.InsertNewline(); return nil })

	// Mode commands
	cmdReg.Register("enter_command_mode", "Enter command mode", 0, 0, func(e *editor.Editor, args []string) error { e.Mode = editor.ModeCommand; return nil })
	cmdReg.Register("cancel_command", "Cancel command mode", 0, 0, func(e *editor.Editor, args []string) error { e.CommandLine = ""; e.Mode = editor.ModeNormal; return nil })
	cmdReg.Register("command_backspace", "Delete last command character", 0, 0, func(e *editor.Editor, args []string) error {
		if len(e.CommandLine) > 0 {
			runes := []rune(e.CommandLine)
			e.CommandLine = string(runes[:len(runes)-1])
		}
		return nil
	})
	cmdReg.Register("enter_visual_mode", "Enter visual mode", 0, 0, func(e *editor.Editor, args []string) error { e.Mode = editor.ModeVisual; e.SetVisualAnchor(); return nil })
	cmdReg.Register("cancel_visual", "Cancel visual mode", 0, 0, func(e *editor.Editor, args []string) error { e.ClearVisual(); return nil })

	// File commands
	cmdReg.Register("w", "Save the current file", 0, 0, func(e *editor.Editor, args []string) error { return e.SaveFile() })
	cmdReg.Register("q", "Quit the editor", 0, 0, func(e *editor.Editor, args []string) error {
		if e.Buffer.Modified {
			return fmt.Errorf("unsaved changes; use :q! to force")
		}
		e.ShouldQuit = true
		return nil
	})
	cmdReg.Register("q!", "Quit without saving", 0, 0, func(e *editor.Editor, args []string) error { e.ShouldQuit = true; return nil })
	cmdReg.Register("wq", "Save and quit", 0, 0, func(e *editor.Editor, args []string) error {
		if err := e.SaveFile(); err != nil {
			return err
		}
		if !e.Buffer.Modified {
			e.ShouldQuit = true
		}
		return nil
	})
	cmdReg.Register("e", "Open a file", 1, 1, func(e *editor.Editor, args []string) error { return e.LoadFile(args[0]) })

	// Undo
	cmdReg.Register("undo", "Undo last change", 0, 0, func(e *editor.Editor, args []string) error { e.Undo(); return nil })

	// Visual commands
	cmdReg.Register("visual_delete", "Delete visual selection", 0, 0, func(e *editor.Editor, args []string) error { e.DeleteSelection(); return nil })
	cmdReg.Register("yank", "Copy selection to clipboard", 0, 0, func(e *editor.Editor, args []string) error { return e.Yank() })
	cmdReg.Register("paste", "Paste clipboard at cursor", 0, 0, func(e *editor.Editor, args []string) error { e.Paste(); return nil })

	// Keymaps
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
