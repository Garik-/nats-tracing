package logger

import "go.uber.org/zap"

var logger = zap.NewNop()

func Get() *zap.Logger {
	return logger
}

func Enable() func() {
	logger, _ = zap.NewDevelopment()
	return func() {
		logger.Sync()
	}
}
