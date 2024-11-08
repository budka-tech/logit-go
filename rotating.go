package logit

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"sync"
	"time"
)

type TimeRotatingWriter struct {
	*lumberjack.Logger
	rotationTime time.Duration
	lastRotation time.Time
	mu           sync.Mutex
}

// NewTimeRotatingWriter создает новый TimeRotatingWriter
func NewTimeRotatingWriter(logger *lumberjack.Logger, rotationTime time.Duration) *TimeRotatingWriter {
	return &TimeRotatingWriter{
		Logger:       logger,
		rotationTime: rotationTime,
		lastRotation: time.Now(),
	}
}

// Write реализует интерфейс io.Writer
func (w *TimeRotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if time.Since(w.lastRotation) >= w.rotationTime {
		if err := w.Logger.Rotate(); err != nil {
			return 0, err
		}
		w.lastRotation = time.Now()
	}

	return w.Logger.Write(p)
}
