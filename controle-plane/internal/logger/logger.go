package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     LogLevel    `json:"level"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

type Logger struct {
	level LogLevel
}

var defaultLogger = &Logger{level: INFO}

func Init(level string) {
	switch level {
	case "DEBUG":
		defaultLogger.level = DEBUG
	case "WARN":
		defaultLogger.level = WARN
	case "ERROR":
		defaultLogger.level = ERROR
	default:
		defaultLogger.level = INFO
	}
}

func log(level LogLevel, requestID, message string, err error, data interface{}) {
	if shouldLog(level) {
		entry := LogEntry{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     level,
			Message:   message,
			RequestID: requestID,
			Data:      data,
		}
		if err != nil {
			entry.Error = err.Error()
		}

		bytes, _ := json.Marshal(entry)
		fmt.Fprintln(os.Stdout, string(bytes))
	}
}

func shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{DEBUG: 0, INFO: 1, WARN: 2, ERROR: 3}
	return levels[level] >= levels[defaultLogger.level]
}

func Debug(requestID, message string, data interface{}) {
	log(DEBUG, requestID, message, nil, data)
}

func Info(requestID, message string, data interface{}) {
	log(INFO, requestID, message, nil, data)
}

func Warn(requestID, message string, err error, data interface{}) {
	log(WARN, requestID, message, err, data)
}

func Error(requestID, message string, err error) {
	log(ERROR, requestID, message, err, nil)
}
