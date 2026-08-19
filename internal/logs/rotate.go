package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type rotatingFile struct {
	dir        string
	maxSize    int64
	maxAge     time.Duration
	maxBackups int

	mu       sync.Mutex
	current  *os.File
	currentN int64
	openedAt time.Time
}

func newRotatingFile(dir string, maxSize int64, maxAge time.Duration, maxBackups int) (*rotatingFile, error) {
	rf := &rotatingFile{
		dir:        dir,
		maxSize:    maxSize,
		maxAge:     maxAge,
		maxBackups: maxBackups,
	}
	if err := rf.open(); err != nil {
		return nil, err
	}
	rf.cleanup()
	return rf, nil
}

func (rf *rotatingFile) Write(p []byte) (int, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.currentN > 0 && rf.currentN+int64(len(p)) > rf.maxSize {
		if err := rf.rotateLocked(); err != nil {
			return 0, err
		}
	}

	n, err := rf.current.Write(p)
	rf.currentN += int64(n)
	return n, err
}

func (rf *rotatingFile) Close() error {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.current == nil {
		return nil
	}
	return rf.current.Close()
}

func (rf *rotatingFile) open() error {
	name := filepath.Join(rf.dir, "proxy.log")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	rf.current = f
	rf.currentN = info.Size()
	rf.openedAt = time.Now()
	return nil
}

func (rf *rotatingFile) rotateLocked() error {
	if rf.current != nil {
		rf.current.Close()
	}

	ts := time.Now().UTC().Format("20060102-150405")
	src := filepath.Join(rf.dir, "proxy.log")
	dst := filepath.Join(rf.dir, fmt.Sprintf("proxy-%s.log", ts))
	if err := os.Rename(src, dst); err != nil && !os.IsNotExist(err) {
		_ = rf.open()
		return err
	}

	rf.cleanup()
	return rf.open()
}

func (rf *rotatingFile) cleanup() {
	if rf.maxBackups <= 0 && rf.maxAge <= 0 {
		return
	}

	entries, err := os.ReadDir(rf.dir)
	if err != nil {
		return
	}

	type backup struct {
		path    string
		modTime time.Time
	}
	var backups []backup
	cutoff := time.Now().Add(-rf.maxAge)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "proxy-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backup{path: filepath.Join(rf.dir, name), modTime: info.ModTime()})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})

	for i, b := range backups {
		shouldRemove := false
		if rf.maxBackups > 0 && i >= rf.maxBackups {
			shouldRemove = true
		}
		if rf.maxAge > 0 && b.modTime.Before(cutoff) {
			shouldRemove = true
		}
		if shouldRemove {
			os.Remove(b.path)
		}
	}
}
