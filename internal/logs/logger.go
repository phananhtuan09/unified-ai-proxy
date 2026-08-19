package logs

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LoggerConfig defines the configuration for the structured logger.
type LoggerConfig struct {
	Dir        string
	Level      slog.Level
	MaxSize    int64
	MaxAge     time.Duration
	MaxBackups int
	RingMax    int
}

func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Dir:        defaultLogDir(),
		Level:      slog.LevelInfo,
		MaxSize:    10 << 20,
		MaxAge:     7 * 24 * time.Hour,
		MaxBackups: 5,
		RingMax:    500,
	}
}

func defaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".logs"
	}
	return filepath.Join(home, ".local", "share", "unified-ai-proxy", "logs")
}

type Logger struct {
	slog   *slog.Logger
	ring   *Store
	file   *rotatingFile
	mu     sync.Mutex
	closed bool
}

func NewLogger(cfg LoggerConfig) (*Logger, error) {
	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return nil, err
	}
	rf, err := newRotatingFile(cfg.Dir, cfg.MaxSize, cfg.MaxAge, cfg.MaxBackups)
	if err != nil {
		return nil, err
	}
	ring := New(cfg.RingMax)
	w := io.MultiWriter(rf, &ringWriter{ring: ring})
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: cfg.Level})
	return &Logger{
		slog: slog.New(handler),
		ring: ring,
		file: rf,
	}, nil
}

func (l *Logger) Slog() *slog.Logger { return l.slog }

func (l *Logger) Ring() *Store { return l.ring }

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	return l.file.Close()
}

type ringWriter struct {
	ring *Store
}

func (w *ringWriter) Write(p []byte) (int, error) {
	n := len(p)
	buf := make([]byte, n)
	copy(buf, p)
	
	s := string(buf)
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	
	w.ring.AddEvent(s)
	return n, nil
}
