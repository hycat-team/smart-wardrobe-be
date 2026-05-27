package logger

import (
	"log"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Interface interface {
	Debug(message string, fields ...zap.Field)
	Info(message string, fields ...zap.Field)
	Warn(message string, fields ...zap.Field)
	Error(message string, fields ...zap.Field)
	Fatal(message string, fields ...zap.Field)
}

type Logger struct {
	zapLogger *zap.Logger
}

func New(env string, logFilePath string, logLevel string, logToFile bool) Interface {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		level = zap.DebugLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	var core zapcore.Core

	consoleWriteSyncer := zapcore.AddSync(os.Stdout)

	var consoleEncoder zapcore.Encoder
	if env == "Development" || env == "dev" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		consoleEncoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	consoleCore := zapcore.NewCore(consoleEncoder, consoleWriteSyncer, level)

	if logToFile {
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
		}

		lumberjackLogger := &lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
			Compress:   true,
		}

		fileWriteSyncer := zapcore.AddSync(lumberjackLogger)

		fileEncoderConfig := encoderConfig
		fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

		fileCore := zapcore.NewCore(fileEncoder, fileWriteSyncer, level)

		core = zapcore.NewTee(consoleCore, fileCore)
	} else {
		core = consoleCore
	}

	zapLogger := zap.New(core, zap.AddCaller())

	return &Logger{
		zapLogger: zapLogger,
	}
}

func (l *Logger) Debug(message string, fields ...zap.Field) {
	l.zapLogger.Debug(message, fields...)
}

func (l *Logger) Info(message string, fields ...zap.Field) {
	l.zapLogger.Info(message, fields...)
}

func (l *Logger) Warn(message string, fields ...zap.Field) {
	l.zapLogger.Warn(message, fields...)
}

func (l *Logger) Error(message string, fields ...zap.Field) {
	l.zapLogger.Error(message, fields...)
}

func (l *Logger) Fatal(message string, fields ...zap.Field) {
	l.zapLogger.Fatal(message, fields...)
}
