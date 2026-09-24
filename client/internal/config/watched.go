package config

import (
	"errors"
	"os"
	"time"
)

// ErrTooLarge is returned by Watched.Read when the last observed size of
// Path exceeded MaxSize.
var ErrTooLarge = errors.New("file too large")

// Watched tracks a config file's mtime and size across calls, so a store can
// tell whether it needs to re-read and re-parse without stat'ing and
// comparing by hand on every call.
type Watched struct {
	Path    string
	MaxSize int64 // 0 means unlimited

	mod  time.Time
	size int64
	seen bool
}

// Changed stats Path and reports whether its mtime or size differ from the
// last call, or from any transition into or out of the file existing at
// all. err is the stat error, if any: on a stat failure, changed reports
// whether Path was successfully seen on a previous call, so a caller can
// tell "just disappeared" (changed=true) from "still missing" (changed=false).
func (w *Watched) Changed() (changed bool, err error) {
	info, err := os.Stat(w.Path)
	if err != nil {
		changed = w.seen
		w.seen = false
		return changed, err
	}
	if w.seen && info.ModTime().Equal(w.mod) && info.Size() == w.size {
		return false, nil
	}
	w.seen = true
	w.mod = info.ModTime()
	w.size = info.Size()
	return true, nil
}

// Seen reports whether the last Changed call found Path present.
func (w *Watched) Seen() bool { return w.seen }

// Size is the size observed by the last successful Changed call.
func (w *Watched) Size() int64 { return w.size }

// Read reads Path, refusing when the size observed by the last Changed call
// exceeds MaxSize.
func (w *Watched) Read() ([]byte, error) {
	if w.MaxSize > 0 && w.size > w.MaxSize {
		return nil, ErrTooLarge
	}
	return os.ReadFile(w.Path)
}

// Sync records Path's current mtime and size as the baseline without
// reading it, so a write the caller just made is not mistaken for an
// external change on the next Changed call.
func (w *Watched) Sync() error {
	info, err := os.Stat(w.Path)
	if err != nil {
		return err
	}
	w.seen = true
	w.mod = info.ModTime()
	w.size = info.Size()
	return nil
}
