package outbound

import (
	"fmt"
	"log"
	"os"
	"time"

	"fsos-server/internal/domain/ports"
)

type FileLogger struct {
	logger    *log.Logger
	component string
	level     ports.LogLevel
}

func NewFileLogger(component string, level ports.LogLevel, logPath string) *FileLogger {
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

	return &FileLogger{
		logger:    log.New(logFile, "", log.LstdFlags),
		component: component,
		level:     level,
	}
}

func (l *FileLogger) log(level ports.LogLevel, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	logMsg := fmt.Sprintf("[%s] [%s] %s", level.String(), l.component, msg)

	if len(fields) > 0 {
		logMsg += " |"
		for key, value := range fields {
			logMsg += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	l.logger.Println(logMsg)
	fmt.Println(logMsg)
}

func (l *FileLogger) Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ports.DEBUG, msg, f)
}

func (l *FileLogger) Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ports.INFO, msg, f)
}

func (l *FileLogger) Warn(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ports.WARN, msg, f)
}

func (l *FileLogger) Error(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ports.ERROR, msg, f)
}

func (l *FileLogger) Fatal(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ports.FATAL, msg, f)
	os.Exit(1)
}
