package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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

	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006/01/02 - 15:04:05"))
	}

	// Set space as the default separator between components
	encoderConfig.ConsoleSeparator = " "

	var core zapcore.Core
	consoleWriteSyncer := zapcore.AddSync(os.Stdout)
	var consoleEncoder zapcore.Encoder

	if env == "Development" || env == "dev" {
		// Wrap the pipe symbol closely around both sides of the level text and preserve the display color
		encoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			var s string
			switch l {
			case zapcore.DebugLevel:
				s = "\x1b[35mDEBUG\x1b[0m"
			case zapcore.InfoLevel:
				s = "\x1b[34mINFO\x1b[0m"
			case zapcore.WarnLevel:
				s = "\x1b[33mWARN\x1b[0m"
			case zapcore.ErrorLevel:
				s = "\x1b[31mERROR\x1b[0m"
			case zapcore.FatalLevel:
				s = "\x1b[31mFATAL\x1b[0m"
			default:
				s = l.CapitalString()
			}
			enc.AppendString(fmt.Sprintf("|%s|", s))
		}
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
		fileEncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("|%s|", l.CapitalString()))
		}
		fileEncoderConfig.ConsoleSeparator = " "
		fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

		fileCore := zapcore.NewCore(fileEncoder, fileWriteSyncer, level)

		core = zapcore.NewTee(consoleCore, fileCore)
	} else {
		core = consoleCore
	}

	zapLogger := zap.New(core)

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
