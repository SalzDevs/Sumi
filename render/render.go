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

// updateScroll adjusts ScrollY so the cursor is always visible.
func updateScroll(e *editor.Editor) {
	vl := visibleLines()
	if e.Cursor.Line < e.Viewport.ScrollY {
		e.Viewport.ScrollY = e.Cursor.Line
	}
	if e.Cursor.Line >= e.Viewport.ScrollY+vl {
		e.Viewport.ScrollY = e.Cursor.Line - vl + 1
	}
	if e.Viewport.ScrollY < 0 {
		e.Viewport.ScrollY = 0
	}
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
