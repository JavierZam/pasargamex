package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
	WarnLogger  *log.Logger // Add warning level
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarnLogger = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func Info(format string, v ...interface{}) {
	InfoLogger.Printf(format, v...)
}

func Error(format string, v ...interface{}) {
	ErrorLogger.Printf(format, v...)
}

func Debug(format string, v ...interface{}) {
	if os.Getenv("ENVIRONMENT") == "development" {
		DebugLogger.Printf(format, v...)
	}
}

func Warn(format string, v ...interface{}) {
	WarnLogger.Printf(format, v...)
}

// Add a context helper for more structured logs
func WithContext(ctx interface{}, format string, v ...interface{}) string {
	_, file, line, _ := runtime.Caller(1)
	contextStr := fmt.Sprintf("%v:%d", file, line)
	if ctx != nil {
		contextStr = fmt.Sprintf("%v - %v", contextStr, ctx)
	}
	return fmt.Sprintf("[%s] %s", contextStr, fmt.Sprintf(format, v...))
}

// Helper for transaction logs
func LogTransactionError(transactionID, action string, err error) {
	Warn("Transaction log error: action=%s, transactionID=%s, error=%v", action, transactionID, err)
}