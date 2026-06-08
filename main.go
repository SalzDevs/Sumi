package main

import (
	"fmt"
	"os"

	"sumi/config"
	"sumi/editor"
	"sumi/lua"
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

// Neovim-style mouse state
var (
	lastClickTime float64
	lastClickLine int
	lastClickCol  int
	clickCount    int
	isMouseDown   bool
	dragStartX    float32
	dragStartY    float32
	isDragging    bool
)

const clickTimeout = 0.4 // seconds, same as Neovim mousetime

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

func handleInput(e *editor.Editor, cmdReg *registry.CommandRegistry, keyReg *registry.KeymapRegistry, font raylib.Font) {
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

	// --- mouse (Neovim-style) ---
	if e.Mode != editor.ModeCommand {
		now := raylib.GetTime()

		// Left click / drag / double-click / triple-click
		if raylib.IsMouseButtonPressed(raylib.MouseLeftButton) {
			pos := raylib.GetMousePosition()
			line, col := render.ClickToLineCol(pos.X, pos.Y, e, font)

			// Detect double/triple click
			isRepeat := (now-lastClickTime < clickTimeout) && (line == lastClickLine) && (col == lastClickCol)
			if isRepeat {
				clickCount++
			} else {
				clickCount = 1
			}
			lastClickTime = now
			lastClickLine = line
			lastClickCol = col

			switch clickCount {
			case 1:
				// Single click: move cursor. In visual mode, this extends selection.
				if e.Mode == editor.ModeVisual {
					e.Cursor.Line = line
					e.Cursor.Col = col
				} else {
					e.Cursor.Line = line
					e.Cursor.Col = col
					e.ClearVisual()
				}
				// Start drag tracking
				isMouseDown = true
				dragStartX = pos.X
				dragStartY = pos.Y
				isDragging = false
			case 2:
				// Double click: select word
				e.SelectWordAt(line, col)
				isMouseDown = true
				dragStartX = pos.X
				dragStartY = pos.Y
				isDragging = false
			case 3:
				// Triple click: select line
				e.SelectLineAt(line)
				clickCount = 0 // reset for next sequence
				isMouseDown = true
				dragStartX = pos.X
				dragStartY = pos.Y
				isDragging = false
			}
		}

		// Drag: extend selection
		if isMouseDown && raylib.IsMouseButtonDown(raylib.MouseLeftButton) {
			pos := raylib.GetMousePosition()
			dx := pos.X - dragStartX
			dy := pos.Y - dragStartY
			if !isDragging && (dx*dx+dy*dy > 9) { // 3px threshold
				isDragging = true
				if e.Mode != editor.ModeVisual {
					// Enter visual mode at drag start position
					line, col := render.ClickToLineCol(dragStartX, dragStartY, e, font)
					e.Cursor.Line = line
					e.Cursor.Col = col
					e.SetVisualAnchor()
					e.Mode = editor.ModeVisual
				}
			}
			if isDragging {
				line, col := render.ClickToLineCol(pos.X, pos.Y, e, font)
				e.Cursor.Line = line
				e.Cursor.Col = col
			}
		}

		if raylib.IsMouseButtonReleased(raylib.MouseLeftButton) {
			isMouseDown = false
			isDragging = false
		}

		// Right click: extend selection to click point
		if raylib.IsMouseButtonPressed(raylib.MouseRightButton) {
			pos := raylib.GetMousePosition()
			line, col := render.ClickToLineCol(pos.X, pos.Y, e, font)
			if e.Mode != editor.ModeVisual {
				e.SetVisualAnchor()
				e.Mode = editor.ModeVisual
			}
			e.Cursor.Line = line
			e.Cursor.Col = col
		}

		// Scroll wheel
		wheel := raylib.GetMouseWheelMove()
		if wheel != 0 {
			e.Viewport.ScrollY -= int(wheel * 3)
			render.ClampScroll(e)
		}
	}

	// --- character input ---
	ch := raylib.GetCharPressed()
	for ch != 0 {
		switch e.Mode {
		case editor.ModeCommand:
			e.CommandLine += string(rune(ch))
		case editor.ModeVisual:
			if ch == 'd' {
				_ = cmdReg.Execute(e, "visual_delete", nil)
			} else if ch == 'y' {
				_ = cmdReg.Execute(e, "yank", nil)
			}
		default: // normal mode
			switch ch {
			case ':':
				_ = cmdReg.Execute(e, "enter_command_mode", nil)
			case 'p':
				_ = cmdReg.Execute(e, "paste", nil)
			default:
				if ch >= 32 && ch < 127 {
					e.InsertChar(rune(ch))
				}
			}
		}
		ch = raylib.GetCharPressed()
	}
}

func main() {
	ed := editor.NewEditor()
	cmdReg := registry.NewCommandRegistry()
	keyReg := registry.NewKeymapRegistry()

	// Register all built-in commands in Go
	config.RegisterBuiltinCommands(cmdReg)

	// Load Lua configuration layer
	bridge := lua.NewBridge(ed, cmdReg, keyReg)
	defer bridge.Close()

	if err := bridge.LoadDefaults(); err != nil {
		fmt.Fprintf(os.Stderr, "Lua default config failed: %v\n", err)
		lua.FallbackKeymaps(keyReg)
	}

	if err := bridge.LoadUserConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Lua user config failed: %v\n", err)
	}

	if len(os.Args) > 1 {
		_ = ed.LoadFile(os.Args[1])
	} else {
		_ = ed.LoadFile("./test.txt")
	}

	raylib.SetConfigFlags(raylib.FlagWindowResizable | raylib.FlagWindowHighdpi)
	raylib.InitWindow(render.DefaultWidth, render.DefaultHeight, "Sumi")
	raylib.SetExitKey(0) // prevent Escape from closing the window
	raylib.SetWindowMinSize(400, 200)
	raylib.SetTargetFPS(60)
	defer raylib.CloseWindow()

	font := raylib.LoadFontEx("assets/JetBrainsMono-Regular.ttf", render.FontSize, nil, 0)
	if font.Texture.ID == 0 {
		font = raylib.GetFontDefault()
	}
	defer raylib.UnloadFont(font)

	for !raylib.WindowShouldClose() && !ed.ShouldQuit {
		handleInput(ed, cmdReg, keyReg, font)
		render.Render(ed, font)
	}
}
