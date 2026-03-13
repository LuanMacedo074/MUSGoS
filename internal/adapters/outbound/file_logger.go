package outbound

import (
	"fmt"
	"log"
	"os"
	"time"

	"fsos-server/internal/domain/ports"
)

type logEntry struct {
	level  ports.LogLevel
	msg    string
	fields map[string]interface{}
	flush  chan struct{} // non-nil = sentinel; drain signals back when caught up
}

type FileLogger struct {
	logger    *log.Logger
	component string
	level     ports.LogLevel
	ch        chan logEntry
}

func NewFileLogger(component string, level ports.LogLevel, logPath string, bufferSize int) *FileLogger {
	if err := os.MkdirAll(logPath, 0755); err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile(
		fmt.Sprintf("%s/%s-%s.log", logPath, component, time.Now().Format("2006-01-02")),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		panic(err)
	}

	fl := &FileLogger{
		logger:    log.New(logFile, "", log.LstdFlags),
		component: component,
		level:     level,
		ch:        make(chan logEntry, bufferSize),
	}

	go fl.drain()

	return fl
}

func (l *FileLogger) drain() {
	for entry := range l.ch {
		if entry.flush != nil {
			close(entry.flush)
			continue
		}
		l.write(entry)
	}
}

func (l *FileLogger) write(entry logEntry) {
	logMsg := fmt.Sprintf("[%s] [%s] %s", entry.level.String(), l.component, entry.msg)

	if len(entry.fields) > 0 {
		logMsg += " |"
		for key, value := range entry.fields {
			logMsg += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	l.logger.Println(logMsg)
	fmt.Println(logMsg)
}

func (l *FileLogger) send(level ports.LogLevel, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	// Copy the map to prevent data races if the caller reuses it.
	var copied map[string]interface{}
	if len(fields) > 0 {
		copied = make(map[string]interface{}, len(fields))
		for k, v := range fields {
			copied[k] = v
		}
	}

	select {
	case l.ch <- logEntry{level: level, msg: msg, fields: copied}:
	default:
		// Buffer full — drop the message and warn on stderr to avoid blocking the caller.
		fmt.Fprintf(os.Stderr, "[DROPPED] log buffer full, message lost: %s\n", msg)
	}
}

func (l *FileLogger) Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.send(ports.DEBUG, msg, f)
}

func (l *FileLogger) Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.send(ports.INFO, msg, f)
}

func (l *FileLogger) Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.send(ports.WARN, msg, f)
}

func (l *FileLogger) Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.send(ports.ERROR, msg, f)
}

func (l *FileLogger) Fatal(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	// Drain pending entries, then write fatal synchronously before exit.
	l.Flush()
	if ports.FATAL >= l.level {
		l.write(logEntry{level: ports.FATAL, msg: msg, fields: f})
	}
	os.Exit(1)
}

// Flush blocks until all pending log entries have been written.
// Safe to call multiple times — does not close the channel.
// Times out after 5 seconds to avoid blocking forever if the buffer is full.
func (l *FileLogger) Flush() {
	done := make(chan struct{})
	select {
	case l.ch <- logEntry{flush: done}:
		<-done
	case <-time.After(5 * time.Second):
		fmt.Fprintf(os.Stderr, "[WARN] flush timeout, some logs may be lost\n")
	}
}
