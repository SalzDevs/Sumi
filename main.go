package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	raylib "github.com/gen2brain/raylib-go/raylib"
)

const (
	fps           = 60
	screenWidth   = 800
	screenHeight  = 400
	gutterWidth   = 20
	fontSize      = 20
	fontSpacing   = 1
)

const (
	modeNormal = iota
	modeCommand
)

// Editor holds all editor state
type Editor struct {
	Buffer      *Buffer
	Cursor      Cursor
	Mode        int
	CommandLine string
	ShouldQuit  bool
}

// Buffer holds the file content
type Buffer struct {
	Lines    []string
	FilePath string
	Modified bool
}

// Cursor position
type Cursor struct {
	Line       int
	Col        int
	DesiredCol int // -1 means unset
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

	// insert after current line
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

func (e *Editor) loadFile(path string) {
	e.Buffer.FilePath = path
	f, err := os.Open(path)
	if err != nil {
		e.Buffer.Lines = []string{""}
		e.Buffer.Modified = false
		return
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// strip trailing \r if CRLF
		if strings.HasSuffix(line, "\r") {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	e.Buffer.Lines = lines
	e.Buffer.Modified = false
	e.Cursor.Line = 0
	e.Cursor.Col = 0
	e.Cursor.DesiredCol = -1
}

func (e *Editor) saveFile() {
	f, err := os.Create(e.Buffer.FilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save file: %v\n", err)
		return
	}
	defer f.Close()
	for _, line := range e.Buffer.Lines {
		fmt.Fprintln(f, line)
	}
	e.Buffer.Modified = false
}

func (e *Editor) executeCommandLine() {
	input := strings.TrimSpace(e.CommandLine)
	if input == "" {
		e.Mode = modeNormal
		return
	}

	parts := strings.Fields(input)
	cmd := parts[0]

	switch cmd {
	case "w":
		e.saveFile()
	case "q":
		e.ShouldQuit = true
	case "wq":
		e.saveFile()
		if !e.Buffer.Modified {
			e.ShouldQuit = true
		}
	case "e":
		if len(parts) > 1 {
			e.loadFile(parts[1])
		}
	}

	e.CommandLine = ""
	e.Mode = modeNormal
}

func main() {
	editor := NewEditor()

	if len(os.Args) > 1 {
		editor.loadFile(os.Args[1])
	} else {
		editor.loadFile("./test.txt")
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
		// --- input ---
		key := raylib.GetKeyPressed()
		for key != 0 {
			if editor.Mode == modeCommand {
				switch key {
				case raylib.KeyEscape:
					editor.CommandLine = ""
					editor.Mode = modeNormal
				case raylib.KeyBackspace:
					if len(editor.CommandLine) > 0 {
						runes := []rune(editor.CommandLine)
						editor.CommandLine = string(runes[:len(runes)-1])
					}
				case raylib.KeyEnter:
					editor.executeCommandLine()
				}
			} else {
				switch key {
				case raylib.KeyRight:
					editor.moveRight()
				case raylib.KeyLeft:
					editor.moveLeft()
				case raylib.KeyDown:
					editor.moveDown()
				case raylib.KeyUp:
					editor.moveUp()
				case raylib.KeyBackspace:
					editor.backspace()
				case raylib.KeyEnter:
					editor.insertNewline()
				}
				if (raylib.IsKeyDown(raylib.KeyLeftControl) || raylib.IsKeyDown(raylib.KeyRightControl)) && key == raylib.KeyS {
					editor.saveFile()
				}
			}
			key = raylib.GetKeyPressed()
		}

		ch := raylib.GetCharPressed()
		for ch != 0 {
			if editor.Mode == modeCommand {
				editor.CommandLine += string(rune(ch))
			} else {
				if ch == ':' {
					editor.Mode = modeCommand
				} else if ch >= 32 && ch < 127 {
					editor.insertChar(rune(ch))
				}
			}
			ch = raylib.GetCharPressed()
		}

		// --- render ---
		raylib.BeginDrawing()
		raylib.ClearBackground(raylib.RayWhite)

		penY := float32(0)
		cursorX := float32(gutterWidth)
		cursorY := float32(0)

		for lineIdx, line := range editor.Buffer.Lines {
			penX := float32(gutterWidth)

			// gutter number
			numStr := fmt.Sprintf("%d", lineIdx+1)
			raylib.DrawTextEx(font, numStr, raylib.Vector2{X: 0, Y: penY}, float32(fontSize), float32(fontSpacing), raylib.Gray)

			runes := []rune(line)
			for col, r := range runes {
				if lineIdx == editor.Cursor.Line && col == editor.Cursor.Col {
					cursorX = penX
					cursorY = penY
				}
				chStr := string(r)
				glyphW := raylib.MeasureTextEx(font, chStr, float32(fontSize), float32(fontSpacing)).X
				raylib.DrawTextEx(font, chStr, raylib.Vector2{X: penX, Y: penY}, float32(fontSize), float32(fontSpacing), raylib.Red)
				penX += glyphW
			}

			if lineIdx == editor.Cursor.Line && editor.Cursor.Col == len(runes) {
				cursorX = penX
				cursorY = penY
			}

			penY += float32(fontSize)
		}

		// cursor
		raylib.DrawRectangle(int32(cursorX), int32(cursorY), 2, fontSize, raylib.Green)

		// command line bar
		if editor.Mode == modeCommand {
			cmdHeight := fontSize + 4
			cmdY := screenHeight - cmdHeight
			raylib.DrawRectangle(0, int32(cmdY), screenWidth, int32(cmdHeight), raylib.LightGray)
			prompt := fmt.Sprintf(":%s", editor.CommandLine)
			raylib.DrawTextEx(font, prompt, raylib.Vector2{X: 4, Y: float32(cmdY) + 2}, float32(fontSize), float32(fontSpacing), raylib.Black)
		}

		raylib.EndDrawing()
	}
}
