package render

import (
	"fmt"
	"strings"
	"time"

	"sumi/editor"
	"sumi/theme"

	raylib "github.com/gen2brain/raylib-go/raylib"
)

// searchMatches returns all start positions of pattern in text.
func searchMatches(text, pattern string) []int {
	if pattern == "" {
		return nil
	}
	var matches []int
	runes := []rune(text)
	strText := string(runes)
	start := 0
	for {
		idx := strings.Index(strText[start:], pattern)
		if idx < 0 {
			break
		}
		matches = append(matches, start+idx)
		start += idx + 1
		if start > len(strText) {
			break
		}
	}
	return matches
}

// -------------------------------------------------------------------------
// Word wrap helpers
// -------------------------------------------------------------------------

// wrapSegments splits text into rune ranges [start, end) that fit within maxWidth.
func wrapSegments(text string, maxWidth float32, font raylib.Font) [][2]int {
	runes := []rune(text)
	if len(runes) == 0 {
		return [][2]int{{0, 0}}
	}
	var segs [][2]int
	segStart := 0
	penX := float32(0)
	for i, r := range runes {
		w := raylib.MeasureTextEx(font, string(r), float32(FontSize), float32(FontSpacing)).X
		if penX+w > maxWidth && i > segStart {
			segs = append(segs, [2]int{segStart, i})
			segStart = i
			penX = 0
		}
		penX += w
	}
	segs = append(segs, [2]int{segStart, len(runes)})
	return segs
}

// widthOfRange returns the pixel width of runes[start:end] within a text string.
func widthOfRange(text string, start, end int, font raylib.Font) float32 {
	runes := []rune(text)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if end <= start {
		return 0
	}
	return raylib.MeasureTextEx(font, string(runes[start:end]), float32(FontSize), float32(FontSpacing)).X
}

// countWrapSegments returns how many display lines a buffer line occupies.
func countWrapSegments(line string, maxWidth float32, font raylib.Font) int {
	if line == "" {
		return 1
	}
	return len(wrapSegments(line, maxWidth, font))
}

const (
	DefaultWidth  = 800
	DefaultHeight = 400
	GutterWidth   = 20
	MinGutter     = 4
	FontSize      = 20
	FontSpacing   = 1
)

func gutterWidth(e *editor.Editor) int {
	if v := e.GetSetting("line_numbers"); v == false {
		return MinGutter
	}
	return GutterWidth
}

var (
	isDrawing   bool
	currentFont raylib.Font
)

func IsDrawing() bool         { return isDrawing }
func ActiveFont() raylib.Font { return currentFont }

func ScreenWidth() int  { return int(raylib.GetScreenWidth()) }
func ScreenHeight() int { return int(raylib.GetScreenHeight()) }

// visibleLines returns how many text lines fit in the content area.
// Never returns less than 1 so we always draw something.
func visibleLines() int {
	statusHeight := FontSize + 4
	h := ScreenHeight()
	if h <= statusHeight {
		return 1
	}
	return (h - statusHeight) / FontSize
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
	h := ScreenHeight()
	if h < statusHeight {
		h = statusHeight
	}
	y := h - statusHeight

	raylib.DrawRectangle(0, int32(y), int32(ScreenWidth()), int32(statusHeight), theme.Get("statusBg"))

	if e.Mode == editor.ModeCommand {
		prompt := fmt.Sprintf(":%s", e.CommandLine)
		raylib.DrawTextEx(font, prompt, raylib.Vector2{X: 4, Y: float32(y) + 2}, float32(FontSize), float32(FontSpacing), theme.Get("statusTxt"))
		return
	}

	// --- status line (normal mode) ---
	var left, right string
	if e.StatusLine != nil {
		left, right = e.StatusLine()
	} else {
		filename := e.Buffer.FilePath
		if filename == "" {
			filename = "[No Name]"
		}
		modified := ""
		if e.Buffer.Modified {
			modified = " [+]"
		}
		left = filename + modified
		modeStr := strings.ToUpper(e.ModeName())
		right = fmt.Sprintf("%d:%d/%d -- %s", e.Cursor.Line+1, e.Cursor.Col+1, len(e.Buffer.Lines), modeStr)
	}

	// Check for active transient error (3-second timeout)
	if e.ErrorMsg != "" && time.Since(e.ErrorTime).Seconds() < 3 {
		right = e.ErrorMsg
	}

	// left side
	raylib.DrawTextEx(font, left, raylib.Vector2{X: 4, Y: float32(y) + 2}, float32(FontSize), float32(FontSpacing), theme.Get("statusTxt"))

	// right side
	rightColor := theme.Get("statusTxt")
	if e.ErrorMsg != "" && time.Since(e.ErrorTime).Seconds() < 3 {
		rightColor = theme.Get("errorTxt")
	}
	rightWidth := raylib.MeasureTextEx(font, right, float32(FontSize), float32(FontSpacing)).X
	raylib.DrawTextEx(font, right, raylib.Vector2{X: float32(ScreenWidth()) - rightWidth - 4, Y: float32(y) + 2}, float32(FontSize), float32(FontSpacing), rightColor)
}

// Render draws the editor state to the screen.
// ClickToLineCol translates screen coordinates to a buffer position.
// Clicking in the gutter clamps to column 0.
// With word wrap enabled, it accounts for wrapped display segments.
func ClickToLineCol(x, y float32, e *editor.Editor, font raylib.Font) (int, int) {
	statusHeight := FontSize + 4
	h := ScreenHeight()
	if h < statusHeight {
		return e.Cursor.Line, e.Cursor.Col
	}
	if y >= float32(h-statusHeight) {
		return e.Cursor.Line, e.Cursor.Col // clicked status bar → ignore
	}

	gw := gutterWidth(e)
	maxW := float32(ScreenWidth() - gw)
	wrapEnabled := e.GetSetting("word_wrap") == true

	// Walk from ScrollY, counting wrap segments, until we find the line+segment at y
	lineIdx := e.Viewport.ScrollY
	remainingY := y
	for lineIdx < len(e.Buffer.Lines) && remainingY >= 0 {
		line := e.Buffer.Lines[lineIdx]
		var segs [][2]int
		if wrapEnabled {
			segs = wrapSegments(line, maxW, font)
		} else {
			segs = [][2]int{{0, len([]rune(line))}}
		}
		segH := float32(len(segs)) * float32(FontSize)
		if remainingY < segH {
			// y falls inside this line; find which segment
			segIdx := int(remainingY / float32(FontSize))
			if segIdx >= len(segs) {
				segIdx = len(segs) - 1
			}
			seg := segs[segIdx]
			targetX := x - float32(gw)
			if targetX <= 0 {
				return lineIdx, seg[0]
			}
			runes := []rune(line)
			penX := float32(0)
			for col := seg[0]; col < seg[1] && col < len(runes); col++ {
				chStr := string(runes[col])
				glyphW := raylib.MeasureTextEx(font, chStr, float32(FontSize), float32(FontSpacing)).X
				if penX+glyphW/2 > targetX {
					return lineIdx, col
				}
				penX += glyphW
			}
			return lineIdx, seg[1]
		}
		remainingY -= segH
		lineIdx++
	}

	// fell off the bottom — clamp to last line
	if lineIdx >= len(e.Buffer.Lines) {
		lineIdx = len(e.Buffer.Lines) - 1
	}
	if lineIdx < 0 {
		lineIdx = 0
	}
	runes := []rune(e.Buffer.Lines[lineIdx])
	return lineIdx, len(runes)
}

func Render(e *editor.Editor, font raylib.Font) {
	updateScroll(e)

	raylib.BeginDrawing()
	raylib.ClearBackground(theme.Get("bg"))

	isDrawing = true
	currentFont = font
	defer func() {
		isDrawing = false
		currentFont = raylib.Font{}
	}()

	statusHeight := FontSize + 4
	contentH := ScreenHeight() - statusHeight
	gw := gutterWidth(e)
	maxW := float32(ScreenWidth() - gw)
	wrapEnabled := e.GetSetting("word_wrap") == true

	penY := float32(0)
	cursorX := float32(gw)
	cursorY := float32(0)
	cursorVisible := false

	lineIdx := e.Viewport.ScrollY
	for lineIdx < len(e.Buffer.Lines) && penY < float32(contentH) {
		line := e.Buffer.Lines[lineIdx]
		runes := []rune(line)

		// compute wrap segments
		var segs [][2]int
		if wrapEnabled {
			segs = wrapSegments(line, maxW, font)
		} else {
			segs = [][2]int{{0, len(runes)}}
		}

		// gutter number (drawn once, at first segment's Y)
		if v := e.GetSetting("line_numbers"); v != false {
			numStr := fmt.Sprintf("%d", lineIdx+1)
			raylib.DrawTextEx(font, numStr, raylib.Vector2{X: 0, Y: penY}, float32(FontSize), float32(FontSpacing), theme.Get("gutter"))
		}

		// cursor-line background for every segment
		if lineIdx == e.Cursor.Line {
			if v := e.GetSetting("cursor_line"); v != false {
				for segIdx, seg := range segs {
					sy := penY + float32(segIdx)*float32(FontSize)
					sw := widthOfRange(line, seg[0], seg[1], font)
					if sw < 1 {
						sw = float32(ScreenWidth() - gw)
					}
					raylib.DrawRectangle(int32(gw), int32(sy), int32(sw)+2, FontSize, theme.Get("cursorLn"))
				}
			}
		}

		// Build per-character color array
		lineColors := make([]raylib.Color, len(runes))
		for i := range lineColors {
			lineColors[i] = theme.Get("text")
		}
		if e.HighlightFn != nil {
			spans := e.HighlightFn(lineIdx, line)
			for _, s := range spans {
				c := raylib.NewColor(
					uint8((s.Color>>24)&0xFF),
					uint8((s.Color>>16)&0xFF),
					uint8((s.Color>>8)&0xFF),
					uint8(s.Color&0xFF),
				)
				for i := s.Start; i <= s.End && i < len(lineColors); i++ {
					if i >= 0 {
						lineColors[i] = c
					}
				}
			}
		}

		// Draw each segment
		for segIdx, seg := range segs {
			segY := penY + float32(segIdx)*float32(FontSize)
			segX := float32(gw)

			// search highlights inside this segment
			if e.SearchPattern != "" {
				for _, matchStart := range searchMatches(line, e.SearchPattern) {
					matchEnd := matchStart + len([]rune(e.SearchPattern))
					if matchEnd > len(runes) {
						matchEnd = len(runes)
					}
					// only draw if overlap with this segment
					if matchEnd <= seg[0] || matchStart >= seg[1] {
						continue
					}
					clipStart := matchStart
					if clipStart < seg[0] {
						clipStart = seg[0]
					}
					clipEnd := matchEnd
					if clipEnd > seg[1] {
						clipEnd = seg[1]
					}
					mx := segX + widthOfRange(line, seg[0], clipStart, font)
					mw := widthOfRange(line, clipStart, clipEnd, font)
					raylib.DrawRectangle(int32(mx), int32(segY), int32(mw), FontSize, theme.Get("searchBg"))
				}
			}

			// characters inside this segment
			for col := seg[0]; col < seg[1] && col < len(runes); col++ {
				if lineIdx == e.Cursor.Line && col == e.Cursor.Col {
					cursorX = segX
					cursorY = segY
					cursorVisible = true
				}
				chStr := string(runes[col])
				glyphW := raylib.MeasureTextEx(font, chStr, float32(FontSize), float32(FontSpacing)).X
				if e.IsSelected(lineIdx, col) {
					raylib.DrawRectangle(int32(segX), int32(segY), int32(glyphW), FontSize, theme.Get("selectBg"))
				}
				raylib.DrawTextEx(font, chStr, raylib.Vector2{X: segX, Y: segY}, float32(FontSize), float32(FontSpacing), lineColors[col])
				segX += glyphW
			}

			if lineIdx == e.Cursor.Line && e.Cursor.Col == len(runes) && segIdx == len(segs)-1 {
				cursorX = segX
				cursorY = segY
				cursorVisible = true
			}
		}

		penY += float32(len(segs)) * float32(FontSize)
		lineIdx++
	}

	// cursor
	if cursorVisible {
		raylib.DrawRectangle(int32(cursorX), int32(cursorY), 2, FontSize, theme.Get("cursor"))
	}

	drawBottomBar(e, font)

	if e.RenderHook != nil {
		e.RenderHook()
	}

	raylib.EndDrawing()
}
