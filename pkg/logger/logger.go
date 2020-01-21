package logger

import (
	"os"
	"path"
	"sync"

	"github.com/spf13/viper"

	"github.com/sirupsen/logrus"
)

// singleton instances
var (
	consoleLogger *logrus.Logger
	fileLogger    *logrus.Logger
)

// for thread safe singleton
var (
	consoleOnce sync.Once
	fileOnce    sync.Once
)

// console logger
func Console() *logrus.Logger {
	consoleOnce.Do(func() {
		consoleLogger = newConsoleLogger()
	})

	return consoleLogger
}

// file logger
func File() *logrus.Logger {
	fileOnce.Do(func() {
		fileLogger = newFileLogger()
	})

	return fileLogger
}

func newConsoleLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	return logger
}

func newFileLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	file, err := os.OpenFile(
		path.Join(viper.GetString("AppRoot"), "/logs/logfile.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		Console().Warn("Failed to logger to file, using default stderr")
	} else {
		logger.SetOutput(file)
	}

	return logger
}
