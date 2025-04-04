package log

import (
	"fmt"
	go_log "log"
	"os"
	"strings"
)

var (
	showLog      = true
	logLevel     = "DEBUG" // 默认日志级别
	stdoutLogger *go_log.Logger
	fileLogger   *go_log.Logger
)

const (
	LOG_LEVEL_DEBUG = "DEBUG"
	LOG_LEVEL_INFO  = "INFO"
	LOG_LEVEL_WARN  = "WARN"
	LOG_LEVEL_ERROR = "ERROR"
	LOG_LEVEL_PANIC = "PANIC"

	LOG_LEVEL_DEBUG_COLOR = "\033[34m"
	LOG_LEVEL_INFO_COLOR  = "\033[32m"
	LOG_LEVEL_WARN_COLOR  = "\033[33m"
	LOG_LEVEL_ERROR_COLOR = "\033[31m"
	LOG_LEVEL_COLOR_END   = "\033[0m"
)

var levels = map[string]int{
	LOG_LEVEL_DEBUG: 1,
	LOG_LEVEL_INFO:  2,
	LOG_LEVEL_WARN:  3,
	LOG_LEVEL_ERROR: 4,
	LOG_LEVEL_PANIC: 5,
}

func init() {
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		logLevel = strings.ToUpper(level)
	}

	if err := os.MkdirAll("logs", os.ModePerm); err != nil {
		panic(fmt.Sprintf("Can't create logs directory: %v", err))
	}

	file, err := os.OpenFile("logs/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("can't open log file: %v", err))
	}

	stdoutLogger = go_log.New(os.Stdout, "", go_log.Ldate|go_log.Ltime|go_log.Lshortfile)
	fileLogger = go_log.New(file, "", go_log.Ldate|go_log.Ltime|go_log.Lshortfile)
}

func shouldLog(level string) bool {
	return showLog && levels[level] >= levels[logLevel]
}

func writeLog(level string, format string, v ...interface{}) {
	if !shouldLog(level) {
		return
	}

	message := fmt.Sprintf("[%s] %s", level, fmt.Sprintf(format, v...))

	// 写入文件（无颜色）
	fileLogger.Output(3, message)

	// 写入标准输出（带颜色）
	var color string
	switch level {
	case LOG_LEVEL_DEBUG:
		color = LOG_LEVEL_DEBUG_COLOR
	case LOG_LEVEL_INFO:
		color = LOG_LEVEL_INFO_COLOR
	case LOG_LEVEL_WARN:
		color = LOG_LEVEL_WARN_COLOR
	case LOG_LEVEL_ERROR, LOG_LEVEL_PANIC:
		color = LOG_LEVEL_ERROR_COLOR
	}
	stdoutLogger.Output(3, color+message+LOG_LEVEL_COLOR_END)

	if level == LOG_LEVEL_PANIC {
		panic(message)
	}
}

func SetShowLog(show bool) {
	showLog = show
}

func Debug(format string, v ...interface{}) {
	writeLog(LOG_LEVEL_DEBUG, format, v...)
}

func Info(format string, v ...interface{}) {
	writeLog(LOG_LEVEL_INFO, format, v...)
}

func Warn(format string, v ...interface{}) {
	writeLog(LOG_LEVEL_WARN, format, v...)
}

func Error(format string, v ...interface{}) {
	writeLog(LOG_LEVEL_ERROR, format, v...)
}

func Panic(format string, v ...interface{}) {
	writeLog(LOG_LEVEL_PANIC, format, v...)
}
