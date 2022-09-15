package logger

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"
	"os"
)

// define available log levels
type Level uint8
const (
	LevelDebug	Level = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelFatal
)

type Logger struct {
	PrintLevel 		Level
	LogLevel		Level
	LogFilePath		string
	PrintSource		bool

	consoleLogger		*log.Logger
	consoleLoggerErr 	*log.Logger
	fileLogger			*log.Logger
	logFile				*os.File
}

var dLogger Logger

func init() {

	dLogger = Logger {
		PrintLevel: LevelDebug,
		LogLevel: LevelInfo,
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
	pc, file, line, ok := runtime.Caller(2)
	if (!ok) {
		file = "#unknown"
		line = 0
	}

	// get the name of the level
	var levelName string
	switch (level) {
		case LevelDebug:	 	levelName = "DEBUG"
		case LevelInfo: 		levelName = "INFO "
		case LevelWarning:		levelName = "WARN "
		case LevelError: 		levelName = "ERROR"
		case LevelFatal:		levelName = "FATAL"
	}

	if (levelName == "") {
		message = fmt.Sprintf("Invalid level value given: %d. Original message: ", level) + message
		levelName = "WARN "
		level = LevelWarning
	}

	printMessage := "[" + levelName + "] " + time.Now().UTC().Format("2006-01-02 03:04:05") + 
		getSourceMessage(file, line, pc, l) + 
		fmt.Sprintf(message, parameters...)

	if (l.LogLevel <= level && l.fileLogger != nil) {
		l.fileLogger.Println(printMessage)
		l.logFile.Sync()

		if (level == LevelFatal) {
			l.CloseFile()
		}
	}

	if (l.PrintLevel <= level) {
		if (level == LevelError) {
			l.consoleLoggerErr.Println(printMessage)
		} else if (level == LevelFatal) {
			l.consoleLoggerErr.Fatal(printMessage)
		} else {
			l.consoleLogger.Println(printMessage)
		}
	}

}

func getSourceMessage(file string, line int, pc uintptr, l Logger) (string) {
	if (!l.PrintSource) {
		return " - "
	}

	fileName := file[strings.LastIndex(file, "/")+1:] + ":" + strconv.Itoa(line)

	return " (" + fileName + ") - "
}

func (l *Logger) setup() {
	l.consoleLogger = log.New(os.Stdout, "", 0)
	l.consoleLoggerErr = log.New(os.Stderr, "", 0)

	if (l.LogFilePath != "") {
		file, err := os.OpenFile(l.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			l.fileLogger = log.New(file, "", 0)
			l.logFile = file
		} else {
			l.Log(LevelError, fmt.Sprintf("Cannot access the log file '%s'\n%s", l.LogFilePath, err.Error()))
		}
	} else {
		l.fileLogger = nil
		if (l.logFile != nil) {
			l.logFile.Close()
			l.logFile = nil
		}
	}
}

func (l *Logger) CloseFile() {
	if (dLogger.logFile != nil) {
		dLogger.logFile.Close()
		dLogger.logFile = nil
		dLogger.fileLogger = nil
	}
}


func SetGlobalLogger(l *Logger) {
	dLogger = *l
	dLogger.setup()
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