package render

import (
	"fmt"

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

func Render(e *editor.Editor, font raylib.Font) {
	raylib.BeginDrawing()
	raylib.ClearBackground(raylib.RayWhite)

	penY := float32(0)
	cursorX := float32(GutterWidth)
	cursorY := float32(0)

	for lineIdx, line := range e.Buffer.Lines {
		penX := float32(GutterWidth)

		numStr := fmt.Sprintf("%d", lineIdx+1)
		raylib.DrawTextEx(font, numStr, raylib.Vector2{X: 0, Y: penY}, float32(FontSize), float32(FontSpacing), raylib.Gray)

		runes := []rune(line)
		for col, r := range runes {
			if lineIdx == e.Cursor.Line && col == e.Cursor.Col {
				cursorX = penX
				cursorY = penY
			}
			chStr := string(r)
			glyphW := raylib.MeasureTextEx(font, chStr, float32(FontSize), float32(FontSpacing)).X
			raylib.DrawTextEx(font, chStr, raylib.Vector2{X: penX, Y: penY}, float32(FontSize), float32(FontSpacing), raylib.Red)
			penX += glyphW
		}

		if lineIdx == e.Cursor.Line && e.Cursor.Col == len(runes) {
			cursorX = penX
			cursorY = penY
		}

		penY += float32(FontSize)
	}

	raylib.DrawRectangle(int32(cursorX), int32(cursorY), 2, FontSize, raylib.Green)

	if e.Mode == editor.ModeCommand {
		cmdHeight := FontSize + 4
		cmdY := ScreenHeight - cmdHeight
		raylib.DrawRectangle(0, int32(cmdY), ScreenWidth, int32(cmdHeight), raylib.LightGray)
		prompt := fmt.Sprintf(":%s", e.CommandLine)
		raylib.DrawTextEx(font, prompt, raylib.Vector2{X: 4, Y: float32(cmdY) + 2}, float32(FontSize), float32(FontSpacing), raylib.Black)
	}

	raylib.EndDrawing()
}
