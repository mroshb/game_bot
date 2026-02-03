package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.SugaredLogger

func Init() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	var level zapcore.Level
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      os.Getenv("APP_ENV") == "development",
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		// Fallback to example logger instead of panicking
		fallbackLogger := zap.NewExample()
		log = fallbackLogger.Sugar()
		log.Warn("Failed to initialize custom logger, using fallback", "error", err)
		return
	}

	log = logger.Sugar()
}

func Debug(msg string, keysAndValues ...interface{}) {
	log.Debugw(msg, keysAndValues...)
}

func Info(msg string, keysAndValues ...interface{}) {
	log.Infow(msg, keysAndValues...)
}

func Warn(msg string, keysAndValues ...interface{}) {
	log.Warnw(msg, keysAndValues...)
}

func Error(msg string, keysAndValues ...interface{}) {
	log.Errorw(msg, keysAndValues...)
}

func Fatal(msg string, err error) {
	log.Fatalw(msg, "error", err)
}

func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}
