package lua

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	glua "github.com/yuin/gopher-lua"
	raylib "github.com/gen2brain/raylib-go/raylib"
	"sumi/editor"
	"sumi/registry"
	"sumi/render"
	"sumi/theme"
)

//go:embed default.lua
var defaultLua string

// Bridge connects the Go engine to the Lua configuration layer.
type Bridge struct {
	L             *glua.LState
	Editor        *editor.Editor
	CmdReg        *registry.CommandRegistry
	KeyReg        *registry.KeymapRegistry
	luaCmds       map[string]*glua.LFunction // command name → Lua handler
	renderHookFn  *glua.LFunction
	statuslineFn  *glua.LFunction
	eventHandlers map[string][]*glua.LFunction
	highlightFn   *glua.LFunction
}

// NewBridge creates a Lua state and exposes the editor API.
func NewBridge(ed *editor.Editor, cmdReg *registry.CommandRegistry, keyReg *registry.KeymapRegistry) *Bridge {
	L := glua.NewState()
	b := &Bridge{
		L:             L,
		Editor:        ed,
		CmdReg:        cmdReg,
		KeyReg:        keyReg,
		luaCmds:       make(map[string]*glua.LFunction),
		eventHandlers: make(map[string][]*glua.LFunction),
	}
	ed.EventDispatcher = b.dispatchEvent
	b.setupRequirePaths()
	b.registerAPI()
	return b
}

func (b *Bridge) setupRequirePaths() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	libDir := filepath.Join(home, ".config", "sumi", "lib")
	pluginDir := filepath.Join(home, ".config", "sumi", "plugins")

	pkg := b.L.GetGlobal("package")
	if pkg == glua.LNil {
		return
	}
	oldPath := string(b.L.GetField(pkg, "path").(glua.LString))
	newPath := oldPath + ";" +
		filepath.Join(libDir, "?.lua") + ";" +
		filepath.Join(libDir, "?", "init.lua") + ";" +
		filepath.Join(pluginDir, "?.lua") + ";" +
		filepath.Join(pluginDir, "?", "init.lua")
	b.L.SetField(pkg, "path", glua.LString(newPath))
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
	b.L.SetField(keys, "F2", glua.LNumber(291))
	b.L.SetField(keys, "F3", glua.LNumber(292))
	b.L.SetField(keys, "F4", glua.LNumber(293))
	b.L.SetField(keys, "F5", glua.LNumber(294))
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
	b.L.SetField(bufTbl, "FilePath", b.L.NewFunction(b.luaBufferFilePath))
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

	// editor settings
	b.L.SetField(edTbl, "SetSetting", b.L.NewFunction(b.luaEditorSetSetting))
	b.L.SetField(edTbl, "GetSetting", b.L.NewFunction(b.luaEditorGetSetting))

	// editor search
	b.L.SetField(edTbl, "SetSearchPattern", b.L.NewFunction(b.luaEditorSetSearchPattern))
	b.L.SetField(edTbl, "SearchPattern", b.L.NewFunction(b.luaEditorSearchPattern))
	b.L.SetField(edTbl, "ClearSearch", b.L.NewFunction(b.luaEditorClearSearch))
	b.L.SetField(edTbl, "FindNext", b.L.NewFunction(b.luaEditorFindNext))
	b.L.SetField(edTbl, "FindPrev", b.L.NewFunction(b.luaEditorFindPrev))

	// editor error display
	b.L.SetField(edTbl, "ShowError", b.L.NewFunction(b.luaEditorShowError))
	b.L.SetField(edTbl, "ClearError", b.L.NewFunction(b.luaEditorClearError))

	// editor tabs
	b.L.SetField(edTbl, "NewTab", b.L.NewFunction(b.luaEditorNewTab))
	b.L.SetField(edTbl, "SwitchTab", b.L.NewFunction(b.luaEditorSwitchTab))
	b.L.SetField(edTbl, "CloseTab", b.L.NewFunction(b.luaEditorCloseTab))
	b.L.SetField(edTbl, "NextTab", b.L.NewFunction(b.luaEditorNextTab))
	b.L.SetField(edTbl, "PrevTab", b.L.NewFunction(b.luaEditorPrevTab))
	b.L.SetField(edTbl, "TabCount", b.L.NewFunction(b.luaEditorTabCount))
	b.L.SetField(edTbl, "TabNames", b.L.NewFunction(b.luaEditorTabNames))
	b.L.SetField(edTbl, "OpenFileInNewTab", b.L.NewFunction(b.luaEditorOpenFileInNewTab))
	b.L.SetField(edTbl, "ActiveTab", b.L.NewFunction(b.luaEditorActiveTab))

	b.L.SetGlobal("editor", edTbl)

	b.registerRenderAPI()
	b.registerThemeAPI()
	b.registerStatuslineAPI()
	b.registerEventsAPI()
	b.registerHighlightAPI()
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
			e.SetMode(editor.ModeNormal)
		case "command":
			e.SetMode(editor.ModeCommand)
		case "visual":
			e.SetMode(editor.ModeVisual)
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
		e.SetMode(editor.ModeVisual)
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
	b.L.SetField(bufTbl, "FilePath", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LString(e.Buffer.FilePath))
		return 1
	}))
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

	// Settings
	b.L.SetField(edTbl, "SetSetting", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		name := L.CheckString(2)
		val := L.Get(3)
		switch v := val.(type) {
		case glua.LBool:
			e.SetSetting(name, bool(v))
		case glua.LNumber:
			n := float64(v)
			if n == float64(int64(n)) {
				e.SetSetting(name, int(n))
			} else {
				e.SetSetting(name, n)
			}
		case glua.LString:
			e.SetSetting(name, string(v))
		case *glua.LNilType:
			e.SetSetting(name, nil)
		default:
			e.SetSetting(name, val)
		}
		return 0
	}))
	b.L.SetField(edTbl, "GetSetting", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		name := L.CheckString(2)
		val := e.GetSetting(name)
		if val == nil {
			L.Push(glua.LNil)
			return 1
		}
		switch v := val.(type) {
		case bool:
			L.Push(glua.LBool(v))
		case int:
			L.Push(glua.LNumber(v))
		case float64:
			L.Push(glua.LNumber(v))
		case string:
			L.Push(glua.LString(v))
		default:
			L.Push(glua.LString(fmt.Sprintf("%v", v)))
		}
		return 1
	}))

	// Search
	b.L.SetField(edTbl, "SetSearchPattern", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.SetSearchPattern(L.CheckString(2))
		return 0
	}))
	b.L.SetField(edTbl, "SearchPattern", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LString(e.SearchPattern))
		return 1
	}))
	b.L.SetField(edTbl, "ClearSearch", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.ClearSearch()
		return 0
	}))
	b.L.SetField(edTbl, "FindNext", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LBool(e.FindNext()))
		return 1
	}))
	b.L.SetField(edTbl, "FindPrev", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LBool(e.FindPrev()))
		return 1
	}))

	// Error display
	b.L.SetField(edTbl, "ShowError", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.ShowError(L.CheckString(2))
		return 0
	}))
	b.L.SetField(edTbl, "ClearError", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		e.ClearError()
		return 0
	}))

	// Tabs
	b.L.SetField(edTbl, "NewTab", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LNumber(e.NewTab() + 1))
		return 1
	}))
	b.L.SetField(edTbl, "SwitchTab", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		idx := L.CheckInt(2) - 1
		L.Push(glua.LBool(e.SwitchTab(idx)))
		return 1
	}))
	b.L.SetField(edTbl, "CloseTab", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		idx := L.CheckInt(2) - 1
		L.Push(glua.LNumber(e.CloseTab(idx) + 1))
		return 1
	}))
	b.L.SetField(edTbl, "NextTab", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LBool(e.NextTab()))
		return 1
	}))
	b.L.SetField(edTbl, "PrevTab", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LBool(e.PrevTab()))
		return 1
	}))
	b.L.SetField(edTbl, "TabCount", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LNumber(len(e.Tabs)))
		return 1
	}))
	b.L.SetField(edTbl, "ActiveTab", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		L.Push(glua.LNumber(e.ActiveTab + 1))
		return 1
	}))
	b.L.SetField(edTbl, "OpenFileInNewTab", b.L.NewFunction(func(L *glua.LState) int {
		_ = L.CheckAny(1)
		path := L.CheckString(2)
		if err := e.OpenFileInNewTab(path); err != nil {
			L.Push(glua.LString(err.Error()))
			return 1
		}
		return 0
	}))

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
		b.Editor.SetMode(editor.ModeNormal)
	case "command":
		b.Editor.SetMode(editor.ModeCommand)
	case "visual":
		b.Editor.SetMode(editor.ModeVisual)
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
	b.Editor.SetMode(editor.ModeVisual)
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

func (b *Bridge) luaBufferFilePath(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LString(b.Editor.Buffer.FilePath))
	return 1
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

func (b *Bridge) luaEditorSetSearchPattern(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.SetSearchPattern(L.CheckString(2))
	return 0
}

func (b *Bridge) luaEditorSearchPattern(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LString(b.Editor.SearchPattern))
	return 1
}

func (b *Bridge) luaEditorClearSearch(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.ClearSearch()
	return 0
}

func (b *Bridge) luaEditorFindNext(L *glua.LState) int {
	_ = L.CheckAny(1)
	found := b.Editor.FindNext()
	L.Push(glua.LBool(found))
	return 1
}

func (b *Bridge) luaEditorFindPrev(L *glua.LState) int {
	_ = L.CheckAny(1)
	found := b.Editor.FindPrev()
	L.Push(glua.LBool(found))
	return 1
}

func (b *Bridge) luaEditorShowError(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.ShowError(L.CheckString(2))
	return 0
}

func (b *Bridge) luaEditorClearError(L *glua.LState) int {
	_ = L.CheckAny(1)
	b.Editor.ClearError()
	return 0
}

func (b *Bridge) luaEditorNewTab(L *glua.LState) int {
	_ = L.CheckAny(1)
	idx := b.Editor.NewTab()
	L.Push(glua.LNumber(idx + 1)) // 1-based for Lua
	return 1
}

func (b *Bridge) luaEditorSwitchTab(L *glua.LState) int {
	_ = L.CheckAny(1)
	idx := L.CheckInt(2) - 1 // to 0-based
	ok := b.Editor.SwitchTab(idx)
	L.Push(glua.LBool(ok))
	return 1
}

func (b *Bridge) luaEditorCloseTab(L *glua.LState) int {
	_ = L.CheckAny(1)
	idx := L.CheckInt(2) - 1 // to 0-based
	newIdx := b.Editor.CloseTab(idx)
	L.Push(glua.LNumber(newIdx + 1))
	return 1
}

func (b *Bridge) luaEditorNextTab(L *glua.LState) int {
	_ = L.CheckAny(1)
	ok := b.Editor.NextTab()
	L.Push(glua.LBool(ok))
	return 1
}

func (b *Bridge) luaEditorPrevTab(L *glua.LState) int {
	_ = L.CheckAny(1)
	ok := b.Editor.PrevTab()
	L.Push(glua.LBool(ok))
	return 1
}

func (b *Bridge) luaEditorTabCount(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(len(b.Editor.Tabs)))
	return 1
}

func (b *Bridge) luaEditorTabNames(L *glua.LState) int {
	_ = L.CheckAny(1)
	tbl := b.L.NewTable()
	for i, t := range b.Editor.Tabs {
		name := t.Buffer.FilePath
		if name == "" {
			name = "[No Name]"
		}
		if t.Buffer.Modified {
			name += " [+]"
		}
		b.L.RawSetInt(tbl, i+1, glua.LString(name))
	}
	b.L.Push(tbl)
	return 1
}

func (b *Bridge) luaEditorOpenFileInNewTab(L *glua.LState) int {
	_ = L.CheckAny(1)
	path := L.CheckString(2)
	if err := b.Editor.OpenFileInNewTab(path); err != nil {
		L.Push(glua.LString(err.Error()))
		return 1
	}
	return 0
}

func (b *Bridge) luaEditorActiveTab(L *glua.LState) int {
	_ = L.CheckAny(1)
	L.Push(glua.LNumber(b.Editor.ActiveTab + 1)) // 1-based for Lua
	return 1
}

func (b *Bridge) luaEditorSetSetting(L *glua.LState) int {
	_ = L.CheckAny(1)
	name := L.CheckString(2)
	val := L.Get(3)

	switch v := val.(type) {
	case glua.LBool:
		b.Editor.SetSetting(name, bool(v))
	case glua.LNumber:
		// store as int if it's a whole number, otherwise float64
		n := float64(v)
		if n == float64(int64(n)) {
			b.Editor.SetSetting(name, int(n))
		} else {
			b.Editor.SetSetting(name, n)
		}
	case glua.LString:
		b.Editor.SetSetting(name, string(v))
	case *glua.LNilType:
		b.Editor.SetSetting(name, nil)
	default:
		b.Editor.SetSetting(name, val)
	}
	return 0
}

func (b *Bridge) luaEditorGetSetting(L *glua.LState) int {
	_ = L.CheckAny(1)
	name := L.CheckString(2)
	val := b.Editor.GetSetting(name)
	if val == nil {
		L.Push(glua.LNil)
		return 1
	}
	switch v := val.(type) {
	case bool:
		L.Push(glua.LBool(v))
	case int:
		L.Push(glua.LNumber(v))
	case float64:
		L.Push(glua.LNumber(v))
	case string:
		L.Push(glua.LString(v))
	default:
		L.Push(glua.LString(fmt.Sprintf("%v", v)))
	}
	return 1
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
// Theme API
// -------------------------------------------------------------------------

func (b *Bridge) registerThemeAPI() {
	themeTbl := b.L.NewTable()
	b.L.SetField(themeTbl, "SetColor", b.L.NewFunction(b.luaThemeSetColor))
	b.L.SetField(themeTbl, "GetColor", b.L.NewFunction(b.luaThemeGetColor))
	b.L.SetField(themeTbl, "Names", b.L.NewFunction(b.luaThemeNames))
	b.L.SetGlobal("theme", themeTbl)
}

func (b *Bridge) luaThemeSetColor(L *glua.LState) int {
	_ = L.CheckAny(1)
	name := L.CheckString(2)
	color := luaColorToRaylib(L.Get(3))
	theme.Set(name, color)
	return 0
}

func (b *Bridge) luaThemeGetColor(L *glua.LState) int {
	_ = L.CheckAny(1)
	name := L.CheckString(2)
	c := theme.Get(name)
	packed := uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
	L.Push(glua.LNumber(packed))
	return 1
}

func (b *Bridge) luaThemeNames(L *glua.LState) int {
	_ = L.CheckAny(1)
	names := theme.Names()
	tbl := b.L.NewTable()
	for i, n := range names {
		b.L.RawSetInt(tbl, i+1, glua.LString(n))
	}
	b.L.Push(tbl)
	return 1
}

// -------------------------------------------------------------------------
// Statusline API
// -------------------------------------------------------------------------

func (b *Bridge) registerStatuslineAPI() {
	slTbl := b.L.NewTable()
	b.L.SetField(slTbl, "Set", b.L.NewFunction(b.luaStatuslineSet))
	b.L.SetGlobal("statusline", slTbl)
}

func (b *Bridge) luaStatuslineSet(L *glua.LState) int {
	_ = L.CheckAny(1)
	fn := L.CheckFunction(2)
	b.statuslineFn = fn

	b.Editor.StatusLine = func() (string, string) {
		if b.statuslineFn == nil {
			return "", ""
		}
		b.L.Push(b.statuslineFn)
		if err := b.L.PCall(0, 2, nil); err != nil {
			fmt.Fprintf(os.Stderr, "statusline error: %v\n", err)
			b.statuslineFn = nil
			b.Editor.StatusLine = nil
			return "", ""
		}
		left := ""
		right := ""
		if ret := b.L.Get(-2); ret != glua.LNil {
			if s, ok := ret.(glua.LString); ok {
				left = string(s)
			}
		}
		if ret := b.L.Get(-1); ret != glua.LNil {
			if s, ok := ret.(glua.LString); ok {
				right = string(s)
			}
		}
		b.L.Pop(2)
		return left, right
	}
	return 0
}

// -------------------------------------------------------------------------
// Events API
// -------------------------------------------------------------------------

func (b *Bridge) registerEventsAPI() {
	eventsTbl := b.L.NewTable()
	b.L.SetField(eventsTbl, "Register", b.L.NewFunction(b.luaEventsRegister))
	b.L.SetField(eventsTbl, "Unregister", b.L.NewFunction(b.luaEventsUnregister))
	b.L.SetGlobal("events", eventsTbl)
}

func (b *Bridge) dispatchEvent(name string, args ...interface{}) {
	handlers := b.eventHandlers[name]
	if len(handlers) == 0 {
		return
	}
	for _, fn := range handlers {
		b.L.Push(fn)
		nArgs := len(args)
		for _, arg := range args {
			switch v := arg.(type) {
			case string:
				b.L.Push(glua.LString(v))
			case int:
				b.L.Push(glua.LNumber(v))
			case bool:
				b.L.Push(glua.LBool(v))
			case float64:
				b.L.Push(glua.LNumber(v))
			default:
				b.L.Push(glua.LString(fmt.Sprintf("%v", v)))
			}
		}
		if err := b.L.PCall(nArgs, 0, nil); err != nil {
			fmt.Fprintf(os.Stderr, "event %s error: %v\n", name, err)
		}
	}
}

func (b *Bridge) luaEventsRegister(L *glua.LState) int {
	_ = L.CheckAny(1)
	name := L.CheckString(2)
	fn := L.CheckFunction(3)
	b.eventHandlers[name] = append(b.eventHandlers[name], fn)
	return 0
}

func (b *Bridge) luaEventsUnregister(L *glua.LState) int {
	_ = L.CheckAny(1)
	name := L.CheckString(2)
	if L.GetTop() >= 3 {
		fn := L.CheckFunction(3)
		filtered := b.eventHandlers[name][:0]
		for _, h := range b.eventHandlers[name] {
			if h != fn {
				filtered = append(filtered, h)
			}
		}
		b.eventHandlers[name] = filtered
	} else {
		delete(b.eventHandlers, name)
	}
	return 0
}

// -------------------------------------------------------------------------
// Highlight API
// -------------------------------------------------------------------------

func (b *Bridge) registerHighlightAPI() {
	hlTbl := b.L.NewTable()
	b.L.SetField(hlTbl, "SetCallback", b.L.NewFunction(b.luaHighlightSetCallback))
	b.L.SetGlobal("highlight", hlTbl)
}

func (b *Bridge) luaHighlightSetCallback(L *glua.LState) int {
	_ = L.CheckAny(1)
	fn := L.CheckFunction(2)
	b.highlightFn = fn

	b.Editor.HighlightFn = func(line int, text string) []editor.HighlightSpan {
		if b.highlightFn == nil {
			return nil
		}
		b.L.Push(b.highlightFn)
		b.L.Push(glua.LNumber(line + 1)) // 1-based for Lua
		b.L.Push(glua.LString(text))
		if err := b.L.PCall(2, 1, nil); err != nil {
			fmt.Fprintf(os.Stderr, "highlight error: %v\n", err)
			b.highlightFn = nil
			b.Editor.HighlightFn = nil
			return nil
		}
		ret := b.L.Get(-1)
		b.L.Pop(1)

		tbl, ok := ret.(*glua.LTable)
		if !ok {
			return nil
		}

		var spans []editor.HighlightSpan
		tbl.ForEach(func(_, value glua.LValue) {
			spanTbl, ok := value.(*glua.LTable)
			if !ok {
				return
			}
			start := 0
			end := 0
			var color uint32

			// support {start=1, end=5, color="#ff0000"} or {1, 5, "#ff0000"}
			if s := b.L.GetField(spanTbl, "start"); s.Type() == glua.LTNumber {
				start = int(s.(glua.LNumber)) - 1 // to 0-based
			} else if s := b.L.RawGetInt(spanTbl, 1); s.Type() == glua.LTNumber {
				start = int(s.(glua.LNumber)) - 1
			}
			if e := b.L.GetField(spanTbl, "end"); e.Type() == glua.LTNumber {
				end = int(e.(glua.LNumber)) - 1
			} else if e := b.L.RawGetInt(spanTbl, 2); e.Type() == glua.LTNumber {
				end = int(e.(glua.LNumber)) - 1
			}
			if c := b.L.GetField(spanTbl, "color"); c.Type() == glua.LTNumber || c.Type() == glua.LTString {
				color = packLuaColor(c)
			} else if c := b.L.RawGetInt(spanTbl, 3); c.Type() == glua.LTNumber || c.Type() == glua.LTString {
				color = packLuaColor(c)
			}
			if end >= start && start >= 0 {
				spans = append(spans, editor.HighlightSpan{Start: start, End: end, Color: color})
			}
		})
		return spans
	}
	return 0
}

func packLuaColor(v glua.LValue) uint32 {
	if n, ok := v.(glua.LNumber); ok {
		return uint32(n)
	}
	s := strings.TrimPrefix(string(v.(glua.LString)), "#")
	if len(s) == 6 {
		if v, err := strconv.ParseUint(s, 16, 32); err == nil {
			return uint32(v)<<8 | 0xFF
		}
	}
	if len(s) == 8 {
		if v, err := strconv.ParseUint(s, 16, 32); err == nil {
			return uint32(v)
		}
	}
	return 0xFFFFFFFF
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

// LoadPlugins scans ~/.config/sumi/plugins/ and executes every .lua file.
// Files are loaded in alphabetical order. Errors are printed to stderr
// but do not stop loading of subsequent plugins.
func (b *Bridge) LoadPlugins() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".config", "sumi", "plugins")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // directory doesn't exist, that's fine
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".lua") {
			continue
		}
		path := filepath.Join(dir, name)
		if err := b.L.DoFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "plugin %s: %v\n", name, err)
		}
	}
}

// Reload re-runs all Lua configuration (defaults, user config, plugins)
// without restarting the editor. Existing commands and keymaps are overwritten.
func (b *Bridge) Reload() error {
	if err := b.LoadDefaults(); err != nil {
		return fmt.Errorf("defaults: %w", err)
	}
	if err := b.LoadUserConfig(); err != nil {
		return fmt.Errorf("user config: %w", err)
	}
	b.LoadPlugins()
	return nil
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
	cmdReg.Register("enter_command_mode", "Enter command mode", 0, 0, func(e *editor.Editor, args []string) error { e.SetMode(editor.ModeCommand); return nil })
	cmdReg.Register("cancel_command", "Cancel command mode", 0, 0, func(e *editor.Editor, args []string) error { e.CommandLine = ""; e.SetMode(editor.ModeNormal); return nil })
	cmdReg.Register("command_backspace", "Delete last command character", 0, 0, func(e *editor.Editor, args []string) error {
		if len(e.CommandLine) > 0 {
			runes := []rune(e.CommandLine)
			e.CommandLine = string(runes[:len(runes)-1])
		}
		return nil
	})
	cmdReg.Register("enter_visual_mode", "Enter visual mode", 0, 0, func(e *editor.Editor, args []string) error { e.SetMode(editor.ModeVisual); e.SetVisualAnchor(); return nil })
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
