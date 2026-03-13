package detector

import (
	"os"
	"strings"
)

func detectTerminal() string {
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram != "" {
		return strings.ToLower(termProgram)
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}

	term := os.Getenv("TERM")
	switch {
	case strings.Contains(term, "kitty"):
		return "kitty"
	case strings.Contains(term, "xterm"):
		return "xterm"
	}

	return term
}
