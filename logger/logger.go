package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//TODO log level cli'ya gore belirlenmeli

func init() {
	encoder := zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())

	logFileName := fmt.Sprintf("gdutils-%s.log", time.Now().Format(time.RFC3339))
	file, _ := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	writerSyncer := zapcore.AddSync(file)

	core := zapcore.NewCore(encoder, writerSyncer, zapcore.DebugLevel)

	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
}

func Info(temp string, s ...interface{}) {
	if temp == "" {
		zap.S().Info(s...)
	} else {
		zap.S().Infof(temp, s...)
	}
}

func Error(temp string, s ...interface{}) {
	if temp == "" {
		zap.S().Error(s...)
	} else {
		zap.S().Errorf(temp, s...)
	}
}

func Debug(temp string, s ...interface{}) {
	if temp == "" {
		zap.S().Debug(s...)
	} else {
		zap.S().Debugf(temp, s...)
	}
}

func Debugw(msg string, keysAndValues ...interface{}) {
	zap.S().Debugw(msg, keysAndValues...)
}

func Panic(temp string, s ...interface{}) {
	if temp == "" {
		zap.S().Panic(s...)
	} else {
		zap.S().Panicf(temp, s...)
	}
}
