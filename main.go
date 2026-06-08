package main

import (
	"os"

	"sumi/config"
	"sumi/editor"
	"sumi/registry"
	"sumi/render"

	raylib "github.com/gen2brain/raylib-go/raylib"
)

const (
	keyDelay  = 30
	keyRepeat = 2
)

var repeatableKeys = map[int32]bool{
	raylib.KeyLeft:      true,
	raylib.KeyRight:     true,
	raylib.KeyUp:        true,
	raylib.KeyDown:      true,
	raylib.KeyBackspace: true,
	raylib.KeyEnter:     true,
}

var keyTimers = make(map[int32]int)

func shouldFire(key int32) bool {
	if !raylib.IsKeyDown(key) {
		keyTimers[key] = 0
		return false
	}
	keyTimers[key]++
	if keyTimers[key] == 1 {
		return true
	}
	if keyTimers[key] > keyDelay && (keyTimers[key]-keyDelay)%keyRepeat == 0 {
		return true
	}
	return false
}

func handleInput(e *editor.Editor, cmdReg *registry.CommandRegistry, keyReg *registry.KeymapRegistry) {
	mode := e.ModeName()

	// --- repeatable keys (arrows, backspace) ---
	for key := range repeatableKeys {
		if shouldFire(key) {
			if cmd, ok := keyReg.Resolve(mode, key); ok {
				_ = cmdReg.Execute(e, cmd, nil)
			}
		}
	}

	// --- one-shot keys (Enter, Escape, etc.) ---
	key := raylib.GetKeyPressed()
	for key != 0 {
		if repeatableKeys[key] {
			key = raylib.GetKeyPressed()
			continue
		}
		if cmd, ok := keyReg.Resolve(mode, key); ok {
			_ = cmdReg.Execute(e, cmd, nil)
		}
		// TODO: chords should eventually live in the keymap too
		ctrl := raylib.IsKeyDown(raylib.KeyLeftControl) || raylib.IsKeyDown(raylib.KeyRightControl)
		cmd := raylib.IsKeyDown(raylib.KeyLeftSuper) || raylib.IsKeyDown(raylib.KeyRightSuper)
		if ctrl && key == raylib.KeyS {
			_ = cmdReg.Execute(e, "w", nil)
		}
		if cmd && key == raylib.KeyR {
			_ = cmdReg.Execute(e, "undo", nil)
		}
		if cmd && key == raylib.KeyV {
			if e.Mode == editor.ModeVisual {
				_ = cmdReg.Execute(e, "cancel_visual", nil)
			} else {
				_ = cmdReg.Execute(e, "enter_visual_mode", nil)
			}
		}
		key = raylib.GetKeyPressed()
	}

	// --- character input ---
	ch := raylib.GetCharPressed()
	for ch != 0 {
		switch e.Mode {
		case editor.ModeCommand:
			e.CommandLine += string(rune(ch))
		case editor.ModeVisual:
			switch ch {
			case 'v':
				_ = cmdReg.Execute(e, "cancel_visual", nil)
			case 'd':
				_ = cmdReg.Execute(e, "visual_delete", nil)
			}
		default: // normal mode
			if ch == ':' {
				_ = cmdReg.Execute(e, "enter_command_mode", nil)
			} else if ch >= 32 && ch < 127 {
				e.InsertChar(rune(ch))
			}
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
