package editor

const (
	ModeNormal = iota
	ModeCommand
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
	}
}

func (e *Editor) ModeName() string {
	switch e.Mode {
	case ModeNormal:
		return "normal"
	case ModeCommand:
		return "command"
	default:
		return "normal"
	}
}

func (e *Editor) ResetDesired() {
	e.Cursor.DesiredCol = -1
}

func (e *Editor) LineLen(line int) int {
	return len([]rune(e.Buffer.Lines[line]))
}
