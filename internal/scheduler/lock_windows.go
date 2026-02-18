//go:build windows

package scheduler

import (
	"errors"
	"os"
)

// FileLock provides a non-blocking file lock on Windows by atomically
// creating a lock file. Creation fails while another process owns the lock.
type FileLock struct {
	path   string
	locked bool
}

// NewFileLock creates a FileLock for the given path.
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// TryLock attempts to acquire the lock without blocking.
// Returns true if the lock was acquired, false if another process holds it.
func (l *FileLock) TryLock() (bool, error) {
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(l.path)
		return false, err
	}
	l.locked = true
	return true, nil
}

// Unlock releases the lock and removes the lock file.
func (l *FileLock) Unlock() error {
	if !l.locked {
		return nil
	}
	if err := os.Remove(l.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	l.locked = false
	return nil
}
