package main

import (
	"rpjosh.de/ncDocConverter/logger"
)

func init() {
	defaultLogger := logger.Logger {
		PrintLevel: 0,
		LogLevel: 1,
		LogFilePath: "log.log",
		PrintSource: true,
	}

	logger.SetGlobalLogger(&defaultLogger)
}

func main() {
	defer logger.CloseFile()


}