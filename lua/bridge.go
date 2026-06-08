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
	L      *glua.LState
	Editor *editor.Editor
	CmdReg *registry.CommandRegistry
	KeyReg *registry.KeymapRegistry
}

// NewBridge creates a Lua state and exposes the editor API.
func NewBridge(ed *editor.Editor, cmdReg *registry.CommandRegistry, keyReg *registry.KeymapRegistry) *Bridge {
	L := glua.NewState()
	b := &Bridge{
		L:      L,
		Editor: ed,
		CmdReg: cmdReg,
		KeyReg: keyReg,
	}
	b.registerAPI()
	return b
}

func (b *Bridge) registerAPI() {
	// keymap:Register(mode, keyCode, commandName)
	km := b.L.NewTable()
	b.L.SetField(km, "Register", b.L.NewFunction(b.luaKeymapRegister))
	b.L.SetGlobal("keymap", km)

	// commands:List() → array of command names
	cmds := b.L.NewTable()
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
	b.L.SetGlobal("keys", keys)

	// editor table (read-only snapshot at load time)
	edTbl := b.L.NewTable()
	b.L.SetField(edTbl, "Mode", glua.LString(b.Editor.ModeName()))
	b.L.SetField(edTbl, "LineCount", glua.LNumber(len(b.Editor.Buffer.Lines)))

	cursorTbl := b.L.NewTable()
	b.L.SetField(cursorTbl, "Line", glua.LNumber(b.Editor.Cursor.Line+1))
	b.L.SetField(cursorTbl, "Col", glua.LNumber(b.Editor.Cursor.Col+1))
	b.L.SetField(edTbl, "Cursor", cursorTbl)

	bufTbl := b.L.NewTable()
	b.L.SetField(bufTbl, "FilePath", glua.LString(b.Editor.Buffer.FilePath))
	b.L.SetField(bufTbl, "Modified", glua.LBool(b.Editor.Buffer.Modified))
	b.L.SetField(edTbl, "Buffer", bufTbl)

	b.L.SetGlobal("editor", edTbl)
}

func (b *Bridge) luaKeymapRegister(L *glua.LState) int {
	_ = L.CheckAny(1) // self (keymap table)
	mode := L.CheckString(2)
	keyCode := L.CheckInt(3)
	cmdName := L.CheckString(4)
	b.KeyReg.Register(mode, int32(keyCode), cmdName)
	return 0
}

func (b *Bridge) luaCommandList(L *glua.LState) int {
	_ = L.CheckAny(1) // self (commands table)
	names := b.CmdReg.Names()
	tbl := b.L.NewTable()
	for i, name := range names {
		b.L.RawSetInt(tbl, i+1, glua.LString(name))
	}
	b.L.Push(tbl)
	return 1
}

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
// Kept here to avoid config package importing raylib in a Lua-only world.
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
