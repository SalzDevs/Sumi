package editor

import (
	"strings"
	"time"
)

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

// Tab holds the per-buffer state for one editor tab.
type Tab struct {
	Buffer        *Buffer
	Cursor        Cursor
	Viewport      Viewport
	UndoStack     UndoStack
	Anchor        LineCol
	Settings      map[string]interface{}
	SearchPattern string
	ErrorMsg      string
	ErrorTime     time.Time
}

type Editor struct {
	// Active view state — mirrors the current tab's data
	Buffer      *Buffer
	Cursor      Cursor
	Mode        int
	CommandLine string
	ShouldQuit  bool
	Viewport    Viewport
	UndoStack   UndoStack
	Anchor      LineCol           // visual mode anchor
	RenderHook  func()            // called during render loop for custom Lua drawing
	StatusLine  func() (string, string) // returns left and right status text
	Settings    map[string]interface{}  // buffer-local settings
	EventDispatcher func(name string, args ...interface{}) // called by core to fire Lua events
	HighlightFn func(line int, text string) []HighlightSpan // syntax coloring; nil means no highlighting
	SearchPattern string            // active search string; empty means no search
	ErrorMsg    string            // transient error message to display
	ErrorTime   time.Time         // when the error was set

	Tabs        []Tab             // all open tabs
	ActiveTab   int               // index of the currently visible tab
}

// HighlightSpan defines a colored rune range on a single line.
// Start and End are 0-based rune indices. End is inclusive.
type HighlightSpan struct {
	Start int
	End   int
	Color uint32 // packed 0xRRGGBBAA
}

// SetSearchPattern stores an active search string.
func (e *Editor) SetSearchPattern(pattern string) {
	e.SearchPattern = pattern
}

// ClearSearch removes the active search.
func (e *Editor) ClearSearch() {
	e.SearchPattern = ""
}

// findMatch searches for the pattern in the buffer starting from a line and column.
// direction: +1 for forward, -1 for backward.
// Returns true if a match was found and the cursor was moved.
func (e *Editor) findMatch(startLine, startCol, direction int) bool {
	if e.SearchPattern == "" {
		return false
	}
	nLines := len(e.Buffer.Lines)
	if nLines == 0 {
		return false
	}

	if direction == 1 {
		// Forward: start from current position, wrap to top if needed
		line := startLine
		col := startCol
		for i := 0; i < nLines+1; i++ {
			if line >= nLines {
				line = 0
				col = 0
			}
			text := e.Buffer.Lines[line]
			runes := []rune(text)
			searchFrom := 0
			if line == startLine && i == 0 {
				searchFrom = col + 1
				if searchFrom > len(runes) {
					searchFrom = len(runes)
				}
			}
			if searchFrom < 0 {
				searchFrom = 0
			}
			if searchFrom <= len(runes) {
				rest := string(runes[searchFrom:])
				idx := strings.Index(rest, e.SearchPattern)
				if idx >= 0 {
					e.Cursor.Line = line
					e.Cursor.Col = searchFrom + idx
					e.ResetDesired()
					return true
				}
			}
			line++
			col = -1 // search from beginning on next line
		}
	} else {
		// Backward: start before current position, wrap to bottom if needed
		line := startLine
		col := startCol
		for i := 0; i < nLines+1; i++ {
			if line < 0 {
				line = nLines - 1
				col = e.LineLen(line)
			}
			text := e.Buffer.Lines[line]
			runes := []rune(text)
			searchEnd := len(runes)
			if line == startLine && i == 0 {
				searchEnd = col
				if searchEnd < 0 {
					searchEnd = 0
				}
			}
			if searchEnd > len(runes) {
				searchEnd = len(runes)
			}
			prefix := string(runes[:searchEnd])
			idx := strings.LastIndex(prefix, e.SearchPattern)
			if idx >= 0 {
				e.Cursor.Line = line
				e.Cursor.Col = idx
				e.ResetDesired()
				return true
			}
			line--
			col = -1
		}
	}
	return false
}

// FindNext moves the cursor to the next search match.
func (e *Editor) FindNext() bool {
	return e.findMatch(e.Cursor.Line, e.Cursor.Col, 1)
}

// FindPrev moves the cursor to the previous search match.
func (e *Editor) FindPrev() bool {
	return e.findMatch(e.Cursor.Line, e.Cursor.Col, -1)
}

// ShowError sets a transient error message to be displayed in the UI.
func (e *Editor) ShowError(msg string) {
	e.ErrorMsg = msg
	e.ErrorTime = time.Now()
}

// ClearError removes the transient error message.
func (e *Editor) ClearError() {
	e.ErrorMsg = ""
}

// -------------------------------------------------------------------------
// Tab management
// -------------------------------------------------------------------------

func NewEditor() *Editor {
	t := Tab{
		Buffer: &Buffer{
			Lines:    []string{""},
			FilePath: "./test.txt",
			Modified: false,
		},
		Cursor:      Cursor{DesiredCol: -1},
		Viewport:    Viewport{ScrollY: 0},
		UndoStack:   UndoStack{pos: -1},
		Anchor:      LineCol{-1, -1},
		Settings:    make(map[string]interface{}),
		SearchPattern: "",
	}

	ed := &Editor{
		Mode:        ModeNormal,
		CommandLine: "",
		ShouldQuit:  false,
		Tabs:        []Tab{t},
		ActiveTab:   0,
	}
	ed.restoreFromTab(0)
	return ed
}

// restoreFromTab copies tab data into the active view fields.
func (e *Editor) restoreFromTab(idx int) {
	if idx < 0 || idx >= len(e.Tabs) {
		return
	}
	t := e.Tabs[idx]
	e.Buffer = t.Buffer
	e.Cursor = t.Cursor
	e.Viewport = t.Viewport
	e.UndoStack = t.UndoStack
	e.Anchor = t.Anchor
	e.Settings = t.Settings
	e.SearchPattern = t.SearchPattern
	e.ErrorMsg = t.ErrorMsg
	e.ErrorTime = t.ErrorTime
	e.ActiveTab = idx
}

// saveToTab copies active view fields back into a tab.
func (e *Editor) saveToTab(idx int) {
	if idx < 0 || idx >= len(e.Tabs) {
		return
	}
	e.Tabs[idx] = Tab{
		Buffer:        e.Buffer,
		Cursor:        e.Cursor,
		Viewport:      e.Viewport,
		UndoStack:     e.UndoStack,
		Anchor:        e.Anchor,
		Settings:      e.Settings,
		SearchPattern: e.SearchPattern,
		ErrorMsg:      e.ErrorMsg,
		ErrorTime:     e.ErrorTime,
	}
}

// SwitchTab changes to a different tab index (0-based).
func (e *Editor) SwitchTab(idx int) bool {
	if idx < 0 || idx >= len(e.Tabs) {
		return false
	}
	if idx == e.ActiveTab {
		return true
	}
	e.saveToTab(e.ActiveTab)
	e.restoreFromTab(idx)
	return true
}

// NewTab creates a blank tab and switches to it.
func (e *Editor) NewTab() int {
	e.saveToTab(e.ActiveTab)
	t := Tab{
		Buffer:      &Buffer{Lines: []string{""}, FilePath: ""},
		Cursor:      Cursor{DesiredCol: -1},
		Viewport:    Viewport{ScrollY: 0},
		UndoStack:   UndoStack{pos: -1},
		Anchor:      LineCol{-1, -1},
		Settings:    make(map[string]interface{}),
		SearchPattern: "",
	}
	e.Tabs = append(e.Tabs, t)
	e.restoreFromTab(len(e.Tabs) - 1)
	return len(e.Tabs) - 1
}

// CloseTab removes a tab. If the last tab is closed, a blank one is created.
// Returns the new active tab index.
func (e *Editor) CloseTab(idx int) int {
	if len(e.Tabs) <= 1 {
		// Replace the only tab with a blank one
		e.Tabs[0] = Tab{
			Buffer:      &Buffer{Lines: []string{""}, FilePath: ""},
			Cursor:      Cursor{DesiredCol: -1},
			Viewport:    Viewport{ScrollY: 0},
			UndoStack:   UndoStack{pos: -1},
			Anchor:      LineCol{-1, -1},
			Settings:    make(map[string]interface{}),
			SearchPattern: "",
		}
		e.restoreFromTab(0)
		return 0
	}

	// Remove the tab at idx
	e.Tabs = append(e.Tabs[:idx], e.Tabs[idx+1:]...)

	// Pick a new active tab
	newIdx := idx
	if newIdx >= len(e.Tabs) {
		newIdx = len(e.Tabs) - 1
	}
	if newIdx < 0 {
		newIdx = 0
	}
	e.restoreFromTab(newIdx)
	return newIdx
}

// NextTab switches to the next tab (wrapping around).
func (e *Editor) NextTab() bool {
	if len(e.Tabs) <= 1 {
		return false
	}
	idx := e.ActiveTab + 1
	if idx >= len(e.Tabs) {
		idx = 0
	}
	return e.SwitchTab(idx)
}

// PrevTab switches to the previous tab (wrapping around).
func (e *Editor) PrevTab() bool {
	if len(e.Tabs) <= 1 {
		return false
	}
	idx := e.ActiveTab - 1
	if idx < 0 {
		idx = len(e.Tabs) - 1
	}
	return e.SwitchTab(idx)
}

// OpenFileInNewTab loads a file into a new tab and switches to it.
func (e *Editor) OpenFileInNewTab(path string) error {
	if err := e.LoadFile(path); err != nil {
		return err
	}
	// Save current tab, create new one with loaded buffer
	e.saveToTab(e.ActiveTab)
	t := Tab{
		Buffer:      e.Buffer,
		Cursor:      Cursor{Line: 0, Col: 0, DesiredCol: -1},
		Viewport:    Viewport{ScrollY: 0},
		UndoStack:   UndoStack{pos: -1},
		Anchor:      LineCol{-1, -1},
		Settings:    make(map[string]interface{}),
		SearchPattern: "",
	}
	e.Tabs = append(e.Tabs, t)
	e.restoreFromTab(len(e.Tabs) - 1)
	return nil
}

// -------------------------------------------------------------------------
// Existing methods (unchanged API surface)
// -------------------------------------------------------------------------

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

// SetMode changes the editor mode and fires the "mode_change" event.
func (e *Editor) SetMode(mode int) {
	if e.Mode == mode {
		return
	}
	e.Mode = mode
	if e.EventDispatcher != nil {
		e.EventDispatcher("mode_change", e.ModeName())
	}
}

// DispatchEvent fires a named event through the dispatcher if present.
func (e *Editor) DispatchEvent(name string, args ...interface{}) {
	if e.EventDispatcher != nil {
		e.EventDispatcher(name, args...)
	}
}

// GetSetting returns a buffer-local setting value, or nil if not set.
func (e *Editor) GetSetting(name string) interface{} {
	if e.Settings == nil {
		return nil
	}
	return e.Settings[name]
}

// SetSetting stores a buffer-local setting value.
func (e *Editor) SetSetting(name string, value interface{}) {
	if e.Settings == nil {
		e.Settings = make(map[string]interface{})
	}
	e.Settings[name] = value
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
	e.SetMode(ModeNormal)
	e.Anchor = LineCol{-1, -1}
}

// SetVisualAnchor sets the anchor to the current cursor position.
func (e *Editor) SetVisualAnchor() {
	e.Anchor = LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}
}

// SelectWordAt enters visual mode and selects the word at (line, col).
// Word chars: [a-zA-Z0-9_]. If on a non-word char, selects that single char.
func (e *Editor) SelectWordAt(line, col int) {
	if line < 0 || line >= len(e.Buffer.Lines) {
		return
	}
	runes := []rune(e.Buffer.Lines[line])
	if len(runes) == 0 {
		e.SetMode(ModeVisual)
		e.Anchor = LineCol{Line: line, Col: 0}
		e.Cursor.Line = line
		e.Cursor.Col = 0
		return
	}
	if col < 0 {
		col = 0
	}
	if col >= len(runes) {
		col = len(runes) - 1
	}

	isWord := func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
	}

	start := col
	for start > 0 && isWord(runes[start-1]) {
		start--
	}
	end := col
	for end < len(runes)-1 && isWord(runes[end+1]) {
		end++
	}

	// If the char at col is not a word char, select just that char
	if !isWord(runes[col]) {
		start = col
		end = col
	}

	e.SetMode(ModeVisual)
	e.Anchor = LineCol{Line: line, Col: start}
	e.Cursor.Line = line
	e.Cursor.Col = end
}

// SelectLineAt enters visual mode and selects the entire line.
func (e *Editor) SelectLineAt(line int) {
	if line < 0 || line >= len(e.Buffer.Lines) {
		return
	}
	runes := []rune(e.Buffer.Lines[line])
	e.SetMode(ModeVisual)
	e.Anchor = LineCol{Line: line, Col: 0}
	e.Cursor.Line = line
	if len(runes) > 0 {
		e.Cursor.Col = len(runes) - 1
	} else {
		e.Cursor.Col = 0
	}
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
		e.DispatchEvent("buffer_change")
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
	e.DispatchEvent("buffer_change")
}
