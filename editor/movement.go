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
	line := []rune(e.Buffer.Lines[e.Cursor.Line])
	before := string(line[:e.Cursor.Col])
	after := string(line[e.Cursor.Col:])
	e.Buffer.Lines[e.Cursor.Line] = before + string(ch) + after
	e.Cursor.Col++
	e.Buffer.Modified = true
	e.ResetDesired()
}

func (e *Editor) InsertNewline() {
	line := []rune(e.Buffer.Lines[e.Cursor.Line])
	before := string(line[:e.Cursor.Col])
	after := string(line[e.Cursor.Col:])
	e.Buffer.Lines[e.Cursor.Line] = before

	e.Buffer.Lines = append(e.Buffer.Lines[:e.Cursor.Line+1], append([]string{after}, e.Buffer.Lines[e.Cursor.Line+1:]...)...)
	e.Cursor.Line++
	e.Cursor.Col = 0
	e.Buffer.Modified = true
	e.ResetDesired()
}

func (e *Editor) Backspace() {
	if e.Cursor.Col > 0 {
		line := []rune(e.Buffer.Lines[e.Cursor.Line])
		e.Buffer.Lines[e.Cursor.Line] = string(line[:e.Cursor.Col-1]) + string(line[e.Cursor.Col:])
		e.Cursor.Col--
		e.Buffer.Modified = true
		e.ResetDesired()
		return
	}
	if e.Cursor.Line == 0 {
		return
	}
	prevLen := e.LineLen(e.Cursor.Line - 1)
	e.Buffer.Lines[e.Cursor.Line-1] += e.Buffer.Lines[e.Cursor.Line]
	e.Buffer.Lines = append(e.Buffer.Lines[:e.Cursor.Line], e.Buffer.Lines[e.Cursor.Line+1:]...)
	e.Cursor.Line--
	e.Cursor.Col = prevLen
	e.Buffer.Modified = true
	e.ResetDesired()
}
