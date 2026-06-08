package main

import (
	"os"

	"sumi/config"
	"sumi/editor"
	"sumi/registry"
	"sumi/render"

	raylib "github.com/gen2brain/raylib-go/raylib"
)

func handleInput(e *editor.Editor, cmdReg *registry.CommandRegistry, keyReg *registry.KeymapRegistry) {
	mode := e.ModeName()

	// --- special keys ---
	key := raylib.GetKeyPressed()
	for key != 0 {
		if cmd, ok := keyReg.Resolve(mode, key); ok {
			_ = cmdReg.Execute(e, cmd, nil)
		}
		// TODO: Ctrl+S chord should eventually live in the keymap too
		if (raylib.IsKeyDown(raylib.KeyLeftControl) || raylib.IsKeyDown(raylib.KeyRightControl)) && key == raylib.KeyS {
			_ = cmdReg.Execute(e, "w", nil)
		}
		key = raylib.GetKeyPressed()
	}

	// --- character input ---
	ch := raylib.GetCharPressed()
	for ch != 0 {
		if e.Mode == editor.ModeCommand {
			e.CommandLine += string(rune(ch))
		} else if ch == ':' {
			_ = cmdReg.Execute(e, "enter_command_mode", nil)
		} else if ch >= 32 && ch < 127 {
			e.InsertChar(rune(ch))
		}
		ch = raylib.GetCharPressed()
	}
}

func main() {
	ed := editor.NewEditor()
	cmdReg := registry.NewCommandRegistry()
	keyReg := registry.NewKeymapRegistry()

	config.RegisterBuiltinCommands(cmdReg)
	config.RegisterBuiltinKeymaps(keyReg)

	if len(os.Args) > 1 {
		_ = ed.LoadFile(os.Args[1])
	} else {
		_ = ed.LoadFile("./test.txt")
	}

	raylib.InitWindow(render.ScreenWidth, render.ScreenHeight, "Sumi")
	raylib.SetTargetFPS(60)
	defer raylib.CloseWindow()

	font := raylib.LoadFontEx("assets/JetBrainsMono-Regular.ttf", render.FontSize, nil, 0)
	if font.Texture.ID == 0 {
		font = raylib.GetFontDefault()
	}
	defer raylib.UnloadFont(font)

	for !raylib.WindowShouldClose() && !ed.ShouldQuit {
		handleInput(ed, cmdReg, keyReg)
		render.Render(ed, font)
	}
}
