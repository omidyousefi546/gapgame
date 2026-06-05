package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

func New(file string) *zap.Logger {

	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "time"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "logs/" + file + ".log",
		MaxSize:    20,
		MaxBackups: 5,
		MaxAge:     14,
		Compress:   true,
	})

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		writer,
		zapcore.InfoLevel,
	)

	return zap.New(core)
}
