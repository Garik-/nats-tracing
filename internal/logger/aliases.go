package logger

import "go.uber.org/zap"

var (
	Error  = zap.Error
	String = zap.String
)
