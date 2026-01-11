package logger

import (
	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Initialize(level string) error {
	var logger *zap.Logger
	var err error

	switch level {
	case "debug":
		logger, err = zap.NewDevelopment()
	default:
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return err
	}

	Log = logger.Sugar()
	return nil
}
