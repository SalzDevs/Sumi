package editor

func (e *Editor) MoveLeft() {
	if e.Cursor.Col > 0 {
		e.Cursor.Col--
	} else if e.Cursor.Line > 0 {
		e.Cursor.Line--
		e.Cursor.Col = e.LineLen(e.Cursor.Line)
	}
	e.ResetDesired()
}

func (e *Editor) MoveRight() {
	if e.Cursor.Col < e.LineLen(e.Cursor.Line) {
		e.Cursor.Col++
	} else if e.Cursor.Line+1 < len(e.Buffer.Lines) {
		e.Cursor.Line++
		e.Cursor.Col = 0
	}
	e.ResetDesired()
}

func (e *Editor) MoveUp() {
	if e.Cursor.Line == 0 {
		return
	}
	if e.Cursor.DesiredCol < 0 {
		e.Cursor.DesiredCol = e.Cursor.Col
	}
	e.Cursor.Line--
	target := e.LineLen(e.Cursor.Line)
	if e.Cursor.DesiredCol < target {
		e.Cursor.Col = e.Cursor.DesiredCol
	} else {
		e.Cursor.Col = target
	}
}

func (e *Editor) MoveDown() {
	if e.Cursor.Line+1 >= len(e.Buffer.Lines) {
		return
	}
	if e.Cursor.DesiredCol < 0 {
		e.Cursor.DesiredCol = e.Cursor.Col
	}
	e.Cursor.Line++
	target := e.LineLen(e.Cursor.Line)
	if e.Cursor.DesiredCol < target {
		e.Cursor.Col = e.Cursor.DesiredCol
	} else {
		e.Cursor.Col = target
	}
}

func (e *Editor) InsertChar(ch rune) {
	cursorBefore := LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}
	oldLine := e.Buffer.Lines[e.Cursor.Line]

	line := []rune(e.Buffer.Lines[e.Cursor.Line])
	before := string(line[:e.Cursor.Col])
	after := string(line[e.Cursor.Col:])
	e.Buffer.Lines[e.Cursor.Line] = before + string(ch) + after
	e.Cursor.Col++
	e.Buffer.Modified = true
	e.ResetDesired()

	e.UndoStack.Push(Edit{
		StartLine:    e.Cursor.Line,
		Before:       []string{oldLine},
		After:        []string{e.Buffer.Lines[e.Cursor.Line]},
		CursorBefore: cursorBefore,
		CursorAfter:  LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col},
	})
}

func (e *Editor) InsertNewline() {
	cursorBefore := LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}
	oldLine := e.Buffer.Lines[e.Cursor.Line]

	line := []rune(e.Buffer.Lines[e.Cursor.Line])
	before := string(line[:e.Cursor.Col])
	after := string(line[e.Cursor.Col:])
	e.Buffer.Lines[e.Cursor.Line] = before

	e.Buffer.Lines = append(e.Buffer.Lines[:e.Cursor.Line+1], append([]string{after}, e.Buffer.Lines[e.Cursor.Line+1:]...)...)
	e.Cursor.Line++
	e.Cursor.Col = 0
	e.Buffer.Modified = true
	e.ResetDesired()

	e.UndoStack.Push(Edit{
		StartLine:    cursorBefore.Line,
		Before:       []string{oldLine},
		After:        []string{before, after},
		CursorBefore: cursorBefore,
		CursorAfter:  LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col},
	})
}

func (e *Editor) Backspace() {
	if e.Cursor.Col > 0 {
		cursorBefore := LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}
		oldLine := e.Buffer.Lines[e.Cursor.Line]

		line := []rune(e.Buffer.Lines[e.Cursor.Line])
		e.Buffer.Lines[e.Cursor.Line] = string(line[:e.Cursor.Col-1]) + string(line[e.Cursor.Col:])
		e.Cursor.Col--
		e.Buffer.Modified = true
		e.ResetDesired()

		e.UndoStack.Push(Edit{
			StartLine:    e.Cursor.Line,
			Before:       []string{oldLine},
			After:        []string{e.Buffer.Lines[e.Cursor.Line]},
			CursorBefore: cursorBefore,
			CursorAfter:  LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col},
		})
		return
	}
	if e.Cursor.Line == 0 {
		return
	}
	cursorBefore := LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col}
	prevLine := e.Buffer.Lines[e.Cursor.Line-1]
	curLine := e.Buffer.Lines[e.Cursor.Line]

	prevLen := e.LineLen(e.Cursor.Line - 1)
	e.Buffer.Lines[e.Cursor.Line-1] += e.Buffer.Lines[e.Cursor.Line]
	e.Buffer.Lines = append(e.Buffer.Lines[:e.Cursor.Line], e.Buffer.Lines[e.Cursor.Line+1:]...)
	e.Cursor.Line--
	e.Cursor.Col = prevLen
	e.Buffer.Modified = true
	e.ResetDesired()

	e.UndoStack.Push(Edit{
		StartLine:    e.Cursor.Line,
		Before:       []string{prevLine, curLine},
		After:        []string{e.Buffer.Lines[e.Cursor.Line]},
		CursorBefore: cursorBefore,
		CursorAfter:  LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col},
	})
}

// applyEdit replaces lines at StartLine with the given slice.
func (e *Editor) applyEdit(lines []string, startLine int) {
	end := startLine + len(lines)
	if end > len(e.Buffer.Lines) {
		end = len(e.Buffer.Lines)
	}
	e.Buffer.Lines = append(e.Buffer.Lines[:startLine], append(lines, e.Buffer.Lines[end:]...)...)
}

func (e *Editor) Undo() {
	edit, ok := e.UndoStack.Undo()
	if !ok {
		return
	}
	e.applyEdit(edit.Before, edit.StartLine)
	e.Cursor.Line = edit.CursorBefore.Line
	e.Cursor.Col = edit.CursorBefore.Col
	e.Buffer.Modified = true
}

// DeleteSelection removes the current visual selection and returns to normal mode.
func (e *Editor) DeleteSelection() {
	if e.Mode != ModeVisual {
		return
	}

	start, end := e.NormalizedSelection()

	// Record undo before mutating
	var before []string
	for i := start.Line; i <= end.Line; i++ {
		before = append(before, e.Buffer.Lines[i])
	}
	cursorBefore := start

	// Perform deletion
	if start.Line == end.Line {
		line := []rune(e.Buffer.Lines[start.Line])
		prefix := string(line[:start.Col])
		suffix := string(line[end.Col+1:])
		e.Buffer.Lines[start.Line] = prefix + suffix
	} else {
		firstLine := []rune(e.Buffer.Lines[start.Line])
		lastLine := []rune(e.Buffer.Lines[end.Line])
		prefix := string(firstLine[:start.Col])
		suffix := string(lastLine[end.Col+1:])
		e.Buffer.Lines[start.Line] = prefix + suffix

		// Remove lines from start.Line+1 through end.Line
		if end.Line+1 < len(e.Buffer.Lines) {
			e.Buffer.Lines = append(e.Buffer.Lines[:start.Line+1], e.Buffer.Lines[end.Line+1:]...)
		} else {
			e.Buffer.Lines = e.Buffer.Lines[:start.Line+1]
		}
	}

	// Clean up empty buffer
	if len(e.Buffer.Lines) == 0 {
		e.Buffer.Lines = []string{""}
	}

	// Ensure cursor lands on a valid line
	if start.Line >= len(e.Buffer.Lines) {
		start.Line = len(e.Buffer.Lines) - 1
	}
	if start.Col > e.LineLen(start.Line) {
		start.Col = e.LineLen(start.Line)
	}

	e.Cursor.Line = start.Line
	e.Cursor.Col = start.Col
	e.ClearVisual()
	e.Buffer.Modified = true

	// Record undo
	e.UndoStack.Push(Edit{
		StartLine:    cursorBefore.Line,
		Before:       before,
		After:        []string{e.Buffer.Lines[start.Line]},
		CursorBefore: cursorBefore,
		CursorAfter:  LineCol{Line: e.Cursor.Line, Col: e.Cursor.Col},
	})
}
