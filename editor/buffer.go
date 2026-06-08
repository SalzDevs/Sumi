package editor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (e *Editor) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "\r") {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}

	e.Buffer.FilePath = path
	e.Buffer.Lines = lines
	e.Buffer.Modified = false
	e.Cursor.Line = 0
	e.Cursor.Col = 0
	e.Cursor.DesiredCol = -1
	return nil
}

func (e *Editor) SaveFile() error {
	f, err := os.Create(e.Buffer.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, line := range e.Buffer.Lines {
		fmt.Fprintln(f, line)
	}
	e.Buffer.Modified = false
	return nil
}
