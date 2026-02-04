package logger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultMaxLogSizeBytes = 2 * 1024 * 1024
	defaultMaxBackups      = 3
	defaultFlushInterval   = time.Second
)

type rotatingWriter struct {
	mu            sync.Mutex
	path          string
	maxSizeBytes  int64
	maxBackups    int
	file          *os.File
	size          int64
	writer        *bufio.Writer
	flushInterval time.Duration
	stopCh        chan struct{}
}

func newRotatingWriter(path string, maxSizeBytes int64, maxBackups int, flushInterval time.Duration) (*rotatingWriter, error) {
	if maxSizeBytes <= 0 {
		maxSizeBytes = defaultMaxLogSizeBytes
	}
	if maxBackups < 1 {
		maxBackups = defaultMaxBackups
	}
	if flushInterval <= 0 {
		flushInterval = defaultFlushInterval
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	w := &rotatingWriter{
		path:          path,
		maxSizeBytes:  maxSizeBytes,
		maxBackups:    maxBackups,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}
	if err := w.openFile(); err != nil {
		return nil, err
	}
	w.startFlushLoop()
	return w, nil
}

func (w *rotatingWriter) openFile() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	w.file = file
	w.size = info.Size()
	w.writer = bufio.NewWriterSize(file, 64*1024)
	return nil
}

func (w *rotatingWriter) startFlushLoop() {
	go func() {
		ticker := time.NewTicker(w.flushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.flush()
			case <-w.stopCh:
				return
			}
		}
	}()
}

func (w *rotatingWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writer != nil {
		_ = w.writer.Flush()
	}
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	if w.size+int64(len(p)) > w.maxSizeBytes {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}

	n, err := w.writer.Write(p)
	w.size += int64(n)
	if err != nil {
		return n, err
	}
	return len(p), nil
}

func (w *rotatingWriter) rotateLocked() error {
	if w.writer != nil {
		_ = w.writer.Flush()
	}
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	// rotate backups
	for i := w.maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", w.path, i)
		newPath := fmt.Sprintf("%s.%d", w.path, i+1)
		_ = os.Rename(oldPath, newPath)
	}
	_ = os.Rename(w.path, fmt.Sprintf("%s.1", w.path))

	w.size = 0
	return w.openFile()
}

func (w *rotatingWriter) Close() error {
	close(w.stopCh)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writer != nil {
		_ = w.writer.Flush()
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// InitFileOutput 将日志输出到文件并启用滚动
func InitFileOutput(path string, maxSizeBytes int64) (io.Writer, error) {
	if path == "" {
		path = filepath.Join("logs", "xunji.log")
	}
	writer, err := newRotatingWriter(path, maxSizeBytes, defaultMaxBackups, defaultFlushInterval)
	if err != nil {
		return nil, err
	}
	SetOutput(writer)
	return writer, nil
}
