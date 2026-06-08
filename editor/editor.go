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
