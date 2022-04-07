package config

import (
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func GetLogger() *zap.Logger {
	if logger != nil {
		return logger
	}
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel), // 输出到控制台
		zapcore.NewCore(encoder, writeSyncer, zapcore.InfoLevel),                 // 写入文件
	)
	logger = zap.New(core, zap.AddCaller())
	return logger
}

func getEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
}

func getLogWriter() zapcore.WriteSyncer {
	now := time.Now()
	lumberJackLogger := &lumberjack.Logger{
		Filename:   Get().LogFile + "." + now.Format("2006-01-02"),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	return zapcore.AddSync(lumberJackLogger)
}
