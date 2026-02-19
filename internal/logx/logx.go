package logx

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

type Logger struct {
	mu    sync.Mutex
	min   Level
	inner *log.Logger
}

func New(min Level) *Logger {
	return &Logger{
		min:   min,
		inner: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) Debug(msg string, kv ...any) { l.log(Debug, msg, kv...) }
func (l *Logger) Info(msg string, kv ...any)  { l.log(Info, msg, kv...) }
func (l *Logger) Warn(msg string, kv ...any)  { l.log(Warn, msg, kv...) }
func (l *Logger) Error(msg string, kv ...any) { l.log(Error, msg, kv...) }

func (l *Logger) log(level Level, msg string, kv ...any) {
	if level < l.min {
		return
	}

	parts := []string{
		time.Now().UTC().Format(time.RFC3339),
		"level=" + strings.ToLower(level.String()),
		"msg=" + quote(msg),
	}

	for i := 0; i+1 < len(kv); i += 2 {
		key := fmt.Sprint(kv[i])
		value := fmt.Sprint(kv[i+1])
		parts = append(parts, key+"="+quote(value))
	}

	l.mu.Lock()
	l.inner.Println(strings.Join(parts, " "))
	l.mu.Unlock()
}

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func quote(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, " \t\n\"") {
		return fmt.Sprintf("%q", value)
	}
	return value
}
