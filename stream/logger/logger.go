package logger

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	Info  *log.Logger
	Error *log.Logger
)

func Setup(logDir string) {
	cwd, err := os.Getwd()
	if err != nil {
		setupFallbackLogging()
		return
	}

	if logDir == "" {
		logDir = filepath.Join(cwd, "stream")
		if filepath.Base(cwd) == "stream" {
			logDir = cwd
		}
	} else if !filepath.IsAbs(logDir) {
		logDir = filepath.Join(cwd, logDir)
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		setupFallbackLogging()
		return
	}

	infoWriter := &dailyFileWriter{dir: logDir, prefix: "app"}
	errWriter := &dailyFileWriter{dir: logDir, prefix: "error"}
	Info = log.New(infoWriter, "INFO ", log.LstdFlags|log.Lmicroseconds)
	Error = log.New(errWriter, "ERROR ", log.LstdFlags|log.Lmicroseconds)
}

func setupFallbackLogging() {
	Info = log.New(os.Stdout, "INFO ", log.LstdFlags|log.Lmicroseconds)
	Error = log.New(os.Stderr, "ERROR ", log.LstdFlags|log.Lmicroseconds)
}

type dailyFileWriter struct {
	dir    string
	prefix string

	mu   sync.Mutex
	date string
	file *os.File
}

func (w *dailyFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if w.file == nil || w.date != today {
		if w.file != nil {
			_ = w.file.Close()
		}
		filename := filepath.Join(w.dir, w.prefix+"-"+today+".log")
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return 0, err
		}
		w.file = f
		w.date = today
	}
	return w.file.Write(p)
}
