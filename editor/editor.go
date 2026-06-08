package editor

import "strings"

const (
	ModeNormal = iota
	ModeCommand
	ModeVisual
)

type Buffer struct {
	Lines    []string
	FilePath string
	Modified bool
}

type Cursor struct {
	Line       int
	Col        int
	DesiredCol int // -1 means unset
}

type LineCol struct {
	Line int
	Col  int
}

type Edit struct {
	StartLine    int
	Before       []string
	After        []string
	CursorBefore LineCol
	CursorAfter  LineCol
}

type UndoStack struct {
	edits []Edit
	pos   int // index of current edit; -1 means nothing to undo
}

func (s *UndoStack) Push(e Edit) {
	// Truncate redo history if we branch from the middle
	if s.pos >= 0 && s.pos < len(s.edits)-1 {
		s.edits = s.edits[:s.pos+1]
	}
	s.edits = append(s.edits, e)
	s.pos = len(s.edits) - 1
}

func (s *UndoStack) Undo() (Edit, bool) {
	if s.pos < 0 {
		return Edit{}, false
	}
	edit := s.edits[s.pos]
	s.pos--
	return edit, true
}



type Viewport struct {
	ScrollY int
}

type Editor struct {
	Buffer      *Buffer
	Cursor      Cursor
	Mode        int
	CommandLine string
	ShouldQuit  bool
	Viewport    Viewport
	UndoStack   UndoStack
	Anchor      LineCol // visual mode anchor
}

func NewEditor() *Editor {
	return &Editor{
		Buffer: &Buffer{
			Lines:    []string{""},
			FilePath: "./test.txt",
			Modified: false,
		},
		Cursor:      Cursor{DesiredCol: -1},
		Mode:        ModeNormal,
		CommandLine: "",
		ShouldQuit:  false,
		Viewport:    Viewport{ScrollY: 0},
		UndoStack:   UndoStack{pos: -1},
		Anchor:      LineCol{-1, -1},
	}
}

func (e *Editor) ModeName() string {
	switch e.Mode {
	case ModeNormal:
		return "normal"
	case ModeCommand:
		return "command"
	case ModeVisual:
		return "visual"
	default:
		return "normal"
	}
}

// NormalizedSelection returns the start and end of the current visual selection,
// ordered so that start <= end. Only valid when Mode == ModeVisual.
func (e *Editor) NormalizedSelection() (LineCol, LineCol) {
	anchor := e.Anchor
	head := LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}

	if anchor.Line < head.Line {
		return anchor, head
	} else if anchor.Line > head.Line {
		return head, anchor
	}
	// same line
	if anchor.Col <= head.Col {
		return anchor, head
	}
	return head, anchor
}

// IsSelected reports whether the given position is inside the visual selection.
func (e *Editor) IsSelected(line, col int) bool {
	if e.Mode != ModeVisual {
		return false
	}
	start, end := e.NormalizedSelection()

	if line < start.Line || line > end.Line {
		return false
	}
	if line == start.Line && line == end.Line {
		return col >= start.Col && col <= end.Col
	}
	if line == start.Line {
		return col >= start.Col
	}
	if line == end.Line {
		return col <= end.Col
	}
	return true
}

// ClearVisual cancels visual mode and clears the anchor.
func (e *Editor) ClearVisual() {
	e.Mode = ModeNormal
	e.Anchor = LineCol{-1, -1}
}

// SetVisualAnchor sets the anchor to the current cursor position.
func (e *Editor) SetVisualAnchor() {
	e.Anchor = LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}
}

func (e *Editor) ResetDesired() {
	e.Cursor.DesiredCol = -1
}

func (e *Editor) LineLen(line int) int {
	return len([]rune(e.Buffer.Lines[line]))
}

// SelectionText returns the currently selected text as a single string.
// Only valid when Mode == ModeVisual.
func (e *Editor) SelectionText() string {
	if e.Mode != ModeVisual {
		return ""
	}
	start, end := e.NormalizedSelection()
	var parts []string
	for line := start.Line; line <= end.Line; line++ {
		runes := []rune(e.Buffer.Lines[line])
		var lineStart, lineEnd int
		if line == start.Line {
			lineStart = start.Col
		}
		if line == end.Line {
			lineEnd = end.Col + 1
		} else {
			lineEnd = len(runes)
		}
		if lineStart < 0 {
			lineStart = 0
		}
		if lineEnd > len(runes) {
			lineEnd = len(runes)
		}
		parts = append(parts, string(runes[lineStart:lineEnd]))
	}
	return strings.Join(parts, "\n")
}

// Yank copies the current selection to the OS clipboard and returns to normal mode.
func (e *Editor) Yank() error {
	text := e.SelectionText()
	if err := SetClipboard(text); err != nil {
		return err
	}
	e.ClearVisual()
	return nil
}

// Paste inserts the OS clipboard at the current cursor position.
// Handles multi-line clipboard content.
func (e *Editor) Paste() {
	text := GetClipboard()
	if text == "" {
		return
	}
	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return
	}

	cursorBefore := LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}
	oldLine := e.Buffer.Lines[e.Cursor.Line]

	// Insert first line at cursor position
	runes := []rune(e.Buffer.Lines[e.Cursor.Line])
	prefix := string(runes[:e.Cursor.Col])
	suffix := string(runes[e.Cursor.Col:])
	first := prefix + lines[0]

	if len(lines) == 1 {
		// Single line paste
		e.Buffer.Lines[e.Cursor.Line] = first + suffix
		e.Cursor.Col = len([]rune(first))
		e.Buffer.Modified = true
		e.UndoStack.Push(Edit{
			StartLine:    cursorBefore.Line,
			Before:       []string{oldLine},
			After:        []string{e.Buffer.Lines[e.Cursor.Line]},
			CursorBefore: cursorBefore,
			CursorAfter:  LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col},
		})
		return
	}

	// Multi-line paste
	lastLine := lines[len(lines)-1] + suffix
	e.Buffer.Lines[e.Cursor.Line] = first

	// Insert middle lines + last line after current line
	newLines := append(lines[1:len(lines)-1], lastLine)
	e.Buffer.Lines = append(e.Buffer.Lines[:e.Cursor.Line+1], append(newLines, e.Buffer.Lines[e.Cursor.Line+1:]...)...)

	e.Cursor.Line += len(lines) - 1
	e.Cursor.Col = len([]rune(lines[len(lines)-1]))
	e.Buffer.Modified = true

	e.UndoStack.Push(Edit{
		StartLine:    cursorBefore.Line,
		Before:       []string{oldLine},
		After:        append([]string{first}, newLines...),
		CursorBefore: cursorBefore,
		CursorAfter:  LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col},
	})
}
