package detector

import (
	"os"
	"path/filepath"
)

type Context struct {
	OSFamily string
	Distro   string
	DEWM     string
	Terminal string
	// SessionBus is false on a headless box, where anything mpris has nothing
	// to talk to and would only log the same failure every tick.
	SessionBus bool
}

func Detect() Context {
	return Context{
		OSFamily:   detectOS(),
		Distro:     detectDistro(),
		DEWM:       detectDEWM(),
		Terminal:   detectTerminal(),
		SessionBus: detectSessionBus(),
	}
}

func detectSessionBus() bool {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return true
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "bus")); err == nil {
			return true
		}
	}
	return false
}
