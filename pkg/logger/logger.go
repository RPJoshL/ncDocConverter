package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Level of the log message
type Level uint8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelFatal
)

type Logger struct {
	PrintLevel  Level
	LogLevel    Level
	LogFilePath string
	PrintSource bool

	consoleLogger    *log.Logger
	consoleLoggerErr *log.Logger
	fileLogger       *log.Logger
	logFile          *os.File
}

var dLogger Logger

func init() {
	dLogger = Logger{
		PrintLevel:  LevelDebug,
		LogLevel:    LevelInfo,
		LogFilePath: "",
		PrintSource: false,
	}

	dLogger.setup()
}

func (l Logger) Log(level Level, message string, parameters ...any) {
	// this function is needed that runtime.Caller(2) is always correct (even on direct call)
	l.log(level, message, parameters...)
}

func (l Logger) log(level Level, message string, parameters ...any) {
	pc, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "#unknown"
		line = 0
	}

	// get the name of the level
	var levelName string
	switch level {
	case LevelDebug:
		levelName = "DEBUG"
	case LevelInfo:
		levelName = "INFO "
	case LevelWarning:
		levelName = "WARN "
	case LevelError:
		levelName = "ERROR"
	case LevelFatal:
		levelName = "FATAL"
	}

	if levelName == "" {
		message = fmt.Sprintf("Invalid level value given: %d. Original message: ", level) + message
		levelName = "WARN "
		level = LevelWarning
	}

	printMessage := "[" + levelName + "] " + time.Now().Local().Format("2006-01-02 15:04:05") +
		getSourceMessage(file, line, pc, l) +
		fmt.Sprintf(message, parameters...)

	if l.LogLevel <= level && l.fileLogger != nil {
		l.fileLogger.Println(printMessage)
		l.logFile.Sync()

		if level == LevelFatal {
			l.CloseFile()
		}
	}

	if l.PrintLevel <= level {
		if level == LevelError {
			l.consoleLoggerErr.Println(printMessage)
		} else if level == LevelFatal {
			l.consoleLoggerErr.Fatal(printMessage)
		} else {
			l.consoleLogger.Println(printMessage)
		}
	}

}

func getSourceMessage(file string, line int, pc uintptr, l Logger) string {
	if !l.PrintSource {
		return " - "
	}

	fileName := file[strings.LastIndex(file, "/")+1:] + ":" + strconv.Itoa(line)

	return " (" + fileName + ") - "
}

func (l *Logger) setup() {
	// log.Ldate|log.Ltime|log.Lshortfile
	l.consoleLogger = log.New(os.Stdout, "", 0)
	l.consoleLoggerErr = log.New(os.Stderr, "", 0)

	if strings.TrimSpace(l.LogFilePath) != "" {
		file, err := os.OpenFile(l.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			l.fileLogger = log.New(file, "", 0)
			l.logFile = file
		} else {
			l.Log(LevelError, fmt.Sprintf("Cannot access the log file '%s'\n%s", l.LogFilePath, err.Error()))
		}
	} else {
		l.fileLogger = nil
		if l.logFile != nil {
			l.logFile.Close()
			l.logFile = nil
		}
	}
}

func (l *Logger) CloseFile() {
	if l.logFile != nil {
		l.logFile.Close()
		l.logFile = nil
		l.fileLogger = nil
	}
}

func SetGlobalLogger(l *Logger) {
	dLogger = *l
	dLogger.setup()
}
func GetGlobalLogger() *Logger {
	return &dLogger
}

func Debug(message string, parameters ...any) {
	dLogger.Log(LevelDebug, message, parameters...)
}
func Info(message string, parameters ...any) {
	dLogger.Log(LevelInfo, message, parameters...)
}
func Warning(message string, parameters ...any) {
	dLogger.Log(LevelWarning, message, parameters...)
}
func Error(message string, parameters ...any) {
	dLogger.Log(LevelError, message, parameters...)
}
func Fatal(message string, parameters ...any) {
	dLogger.Log(LevelFatal, message, parameters...)
}

func CloseFile() {
	dLogger.CloseFile()
}

// Tries to convert the given level name to the corresponding level code.
// Allowed values are: 'debug', 'info', 'warn', 'warning', 'error', 'panic' and 'fatal'
// If an incorrect level name was given an warning is logged and info will be returned
func GetLevelByName(levelName string) Level {
	levelName = strings.ToLower(levelName)
	switch levelName {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarning
	case "error":
		return LevelError
	case "panic", "fatal":
		return LevelFatal

	default:
		{
			Warning("Unable to parse the level name '%s'. Expected 'debug', 'info', 'warn', 'error' or 'fatal'", levelName)
			return LevelInfo
		}
	}
}
