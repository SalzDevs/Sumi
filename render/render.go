package render

import (
	"fmt"
	"strings"

	"sumi/editor"

	raylib "github.com/gen2brain/raylib-go/raylib"
)

const (
	ScreenWidth  = 800
	ScreenHeight = 400
	GutterWidth  = 20
	FontSize     = 20
	FontSpacing  = 1
)

// visibleLines returns how many text lines fit in the content area.
func visibleLines() int {
	statusHeight := FontSize + 4
	contentHeight := ScreenHeight - statusHeight
	return contentHeight / FontSize
}

func maxScroll(e *editor.Editor) int {
	vl := visibleLines()
	ms := len(e.Buffer.Lines) - vl
	if ms < 0 {
		ms = 0
	}
	return ms
}

// ClampScroll ensures ScrollY stays within valid bounds.
func ClampScroll(e *editor.Editor) {
	if e.Viewport.ScrollY < 0 {
		e.Viewport.ScrollY = 0
	}
	if e.Viewport.ScrollY > maxScroll(e) {
		e.Viewport.ScrollY = maxScroll(e)
	}
}

// updateScroll adjusts ScrollY so the cursor is always visible.
func updateScroll(e *editor.Editor) {
	vl := visibleLines()
	if e.Cursor.Line < e.Viewport.ScrollY {
		e.Viewport.ScrollY = e.Cursor.Line
	}
	if e.Cursor.Line >= e.Viewport.ScrollY+vl {
		e.Viewport.ScrollY = e.Cursor.Line - vl + 1
	}
	ClampScroll(e)
}

// drawBottomBar renders the status line (normal mode) or command bar (command mode).
func drawBottomBar(e *editor.Editor, font raylib.Font) {
	statusHeight := FontSize + 4
	y := ScreenHeight - statusHeight

	raylib.DrawRectangle(0, int32(y), ScreenWidth, int32(statusHeight), raylib.LightGray)

	if e.Mode == editor.ModeCommand {
		prompt := fmt.Sprintf(":%s", e.CommandLine)
		raylib.DrawTextEx(font, prompt, raylib.Vector2{X: 4, Y: float32(y) + 2}, float32(FontSize), float32(FontSpacing), raylib.Black)
		return
	}

	// --- status line (normal mode) ---
	filename := e.Buffer.FilePath
	if filename == "" {
		filename = "[No Name]"
	}
	modified := ""
	if e.Buffer.Modified {
		modified = " [+]"
	}
	left := filename + modified

	modeStr := strings.ToUpper(e.ModeName())
	right := fmt.Sprintf("%d:%d/%d -- %s", e.Cursor.Line+1, e.Cursor.Col+1, len(e.Buffer.Lines), modeStr)

	// left side
	raylib.DrawTextEx(font, left, raylib.Vector2{X: 4, Y: float32(y) + 2}, float32(FontSize), float32(FontSpacing), raylib.Black)

	// right side
	rightWidth := raylib.MeasureTextEx(font, right, float32(FontSize), float32(FontSpacing)).X
	raylib.DrawTextEx(font, right, raylib.Vector2{X: float32(ScreenWidth) - rightWidth - 4, Y: float32(y) + 2}, float32(FontSize), float32(FontSpacing), raylib.Black)
}

// Render draws the editor state to the screen.
// ClickToLineCol translates screen coordinates to a buffer position.
// Clicking in the gutter clamps to column 0.
func ClickToLineCol(x, y float32, e *editor.Editor, font raylib.Font) (int, int) {
	statusHeight := FontSize + 4
	if y >= float32(ScreenHeight-statusHeight) {
		return e.Cursor.Line, e.Cursor.Col // clicked status bar → ignore
	}

	line := e.Viewport.ScrollY + int(y/FontSize)
	if line >= len(e.Buffer.Lines) {
		line = len(e.Buffer.Lines) - 1
	}
	if line < 0 {
		line = 0
	}

	targetX := x - GutterWidth
	if targetX <= 0 {
		return line, 0
	}

	runes := []rune(e.Buffer.Lines[line])
	penX := float32(0)
	for col, r := range runes {
		chStr := string(r)
		glyphW := raylib.MeasureTextEx(font, chStr, float32(FontSize), float32(FontSpacing)).X
		if penX+glyphW/2 > targetX {
			return line, col
		}
		penX += glyphW
	}
	return line, len(runes)
}

func Render(e *editor.Editor, font raylib.Font) {
	updateScroll(e)

	raylib.BeginDrawing()
	raylib.ClearBackground(raylib.RayWhite)

	vl := visibleLines()
	startLine := e.Viewport.ScrollY
	endLine := startLine + vl
	if endLine > len(e.Buffer.Lines) {
		endLine = len(e.Buffer.Lines)
	}

	penY := float32(0)
	cursorX := float32(GutterWidth)
	cursorY := float32(0)
	cursorVisible := false

	for lineIdx := startLine; lineIdx < endLine; lineIdx++ {
		line := e.Buffer.Lines[lineIdx]
		penX := float32(GutterWidth)

		// gutter number
		numStr := fmt.Sprintf("%d", lineIdx+1)
		raylib.DrawTextEx(font, numStr, raylib.Vector2{X: 0, Y: penY}, float32(FontSize), float32(FontSpacing), raylib.Gray)

		runes := []rune(line)
		for col, r := range runes {
			if lineIdx == e.Cursor.Line && col == e.Cursor.Col {
				cursorX = penX
				cursorY = penY
				cursorVisible = true
			}
			chStr := string(r)
			glyphW := raylib.MeasureTextEx(font, chStr, float32(FontSize), float32(FontSpacing)).X
			if e.IsSelected(lineIdx, col) {
				raylib.DrawRectangle(int32(penX), int32(penY), int32(glyphW), FontSize, raylib.SkyBlue)
			}
			raylib.DrawTextEx(font, chStr, raylib.Vector2{X: penX, Y: penY}, float32(FontSize), float32(FontSpacing), raylib.Red)
			penX += glyphW
		}

		if lineIdx == e.Cursor.Line && e.Cursor.Col == len(runes) {
			cursorX = penX
			cursorY = penY
			cursorVisible = true
		}

		penY += float32(FontSize)
	}

	// cursor
	if cursorVisible {
		raylib.DrawRectangle(int32(cursorX), int32(cursorY), 2, FontSize, raylib.Green)
	}

	drawBottomBar(e, font)

	raylib.EndDrawing()
}
