package monitor

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	warnLogger  *log.Logger
)

const (
	LogLevelInfo  = "INFO"
	LogLevelError = "ERROR"
	LogLevelWarn  = "WARN"
)

// init 函数用于初始化日志器
func init() {
	infoLogger = log.New(os.Stdout, "[INFO]  ", log.Lshortfile)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.Lshortfile)
	warnLogger = log.New(os.Stdout, "[WARN]  ", log.Lshortfile)
}

// FmtLog 格式化日志消息并调用相应的日志器。
func FmtLog(level string, format string, args ...interface{}) {
	timestamp := time.Now().Format(time.RFC3339) // ISO 8601 format
	message := fmt.Sprintf(format, args...)
	switch level {
	case LogLevelInfo:
		infoLogger.Printf("%s - %s", timestamp, message)
	case LogLevelError:
		errorLogger.Printf("%s - %s", timestamp, message)
	case LogLevelWarn:
		warnLogger.Printf("%s - %s", timestamp, message)
	}
}