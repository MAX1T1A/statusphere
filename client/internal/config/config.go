package config

import (
	"os"
	"path/filepath"
	"sync/atomic"
)

const AppName = "statusphere"

var baseDir atomic.Pointer[string]

func SetDir(dir string) {
	baseDir.Store(&dir)
}

func Dir() string {
	if dir := baseDir.Load(); dir != nil && *dir != "" {
		return *dir
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, AppName)
}

func File(name string) string {
	return filepath.Join(Dir(), name)
}

func CacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, AppName)
}

func LogPath() string {
	return filepath.Join(CacheDir(), AppName+".log")
}

func Write(name string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(File(name), data, perm)
}

func Read(name string) ([]byte, error) {
	return os.ReadFile(File(name))
}
