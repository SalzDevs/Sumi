package editor

import (
	"github.com/atotto/clipboard"
)

// GetClipboard returns the OS clipboard contents, or empty string on error.
func GetClipboard() string {
	s, err := clipboard.ReadAll()
	if err != nil {
		return ""
	}
	return s
}

// SetClipboard writes text to the OS clipboard.
func SetClipboard(text string) error {
	return clipboard.WriteAll(text)
}
