package logger

import (
    "fmt"
    "log"
    "os"
    "time"
)

type LogLevel int

const (
    DEBUG LogLevel = iota
    INFO
    WARN
    ERROR
    FATAL
)

func (l LogLevel) String() string {
    switch l {
    case DEBUG:
        return "DEBUG"
    case INFO:
        return "INFO"
    case WARN:
        return "WARN"
    case ERROR:
        return "ERROR"
    case FATAL:
        return "FATAL"
    default:
        return "UNKNOWN"
    }
}

type Logger struct {
    logger    *log.Logger
    component string
    level     LogLevel
}

func New(component string, level LogLevel) *Logger {
    // Create logs directory if it doesn't exist
    if err := os.MkdirAll("logs", 0755); err != nil {
        panic(err)
    }

    // Create log file with timestamp
    logFile, err := os.OpenFile(
        fmt.Sprintf("logs/%s-%s.log", component, time.Now().Format("2006-01-02")),
        os.O_CREATE|os.O_WRONLY|os.O_APPEND,
        0666,
    )
    if err != nil {
        panic(err)
    }

    logger := log.New(logFile, "", log.LstdFlags)
    
    return &Logger{
        logger:    logger,
        component: component,
        level:     level,
    }
}

func (l *Logger) log(level LogLevel, msg string, fields map[string]interface{}) {
    if level < l.level {
        return
    }

    logMsg := fmt.Sprintf("[%s] [%s] %s", level.String(), l.component, msg)
    
    if fields != nil && len(fields) > 0 {
        logMsg += " |"
        for key, value := range fields {
            logMsg += fmt.Sprintf(" %s=%v", key, value)
        }
    }

    l.logger.Println(logMsg)
    fmt.Println(logMsg) // Also print to console
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
    var f map[string]interface{}
    if len(fields) > 0 {
        f = fields[0]
    }
    l.log(DEBUG, msg, f)
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
    var f map[string]interface{}
    if len(fields) > 0 {
        f = fields[0]
    }
    l.log(INFO, msg, f)
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
    var f map[string]interface{}
    if len(fields) > 0 {
        f = fields[0]
    }
    l.log(WARN, msg, f)
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
    var f map[string]interface{}
    if len(fields) > 0 {
        f = fields[0]
    }
    l.log(ERROR, msg, f)
}

func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
    var f map[string]interface{}
    if len(fields) > 0 {
        f = fields[0]
    }
    l.log(FATAL, msg, f)
    os.Exit(1)
}

// Raw prints directly to console and log file without formatting
// Useful for hexdump output
func (l *Logger) Raw(content string) {
    l.logger.Print(content)
    fmt.Print(content)
}

// Protocol-specific helper methods
func (l *Logger) LogConnection(clientIP string) {
    l.Info("New connection", map[string]interface{}{
        "type":      "CONNECTION",
        "client_ip": clientIP,
    })
}

func (l *Logger) LogMessage(from, subject, content string) {
    // Truncate content for readability
    truncated := content
    if len(content) > 100 {
        truncated = content[:100] + "..."
    }
    
    l.Info("Message received", map[string]interface{}{
        "type":    "MESSAGE",
        "from":    from,
        "subject": subject,
        "content": truncated,
    })
}

func (l *Logger) LogDisconnection(playerName, reason string) {
    l.Info("Player disconnected", map[string]interface{}{
        "type":   "DISCONNECTION",
        "player": playerName,
        "reason": reason,
    })
}

func (l *Logger) LogHeartbeat(playerName string) {
    l.Debug("Heartbeat received", map[string]interface{}{
        "type":   "HEARTBEAT",
        "player": playerName,
    })
}

func (l *Logger) LogGameTick(processedMessages int) {
    l.Debug("Game tick processed", map[string]interface{}{
        "type":      "GAME_TICK",
        "processed": processedMessages,
    })
}

func (l *Logger) LogChatMessage(playerName, content string) {
    l.Info("Chat message", map[string]interface{}{
        "type":    "CHAT",
        "player":  playerName,
        "content": content,
    })
}

func (l *Logger) LogCharacterCreation(username string, success bool) {
    l.Info("Character creation", map[string]interface{}{
        "type":     "CHARACTER_CREATE",
        "username": username,
        "success":  success,
    })
}

func (l *Logger) LogCharacterLoad(username string, success bool) {
    l.Info("Character load", map[string]interface{}{
        "type":     "CHARACTER_LOAD",
        "username": username,
        "success":  success,
    })
}

// TCP-specific logging methods
func (l *Logger) LogTCPPacket(clientIP string, offset int, bytes int) {
    l.Info("TCP Packet", map[string]interface{}{
        "client": clientIP,
        "offset": fmt.Sprintf("0x%04X", offset),
        "bytes":  bytes,
    })
}

func (l *Logger) LogHexDump(hexdump string) {
    // Log hexdump with a separator for clarity
    l.Raw("--- TCP Hexdump ---\n")
    l.Raw(hexdump)
    l.Raw("--- End Hexdump ---\n")
}