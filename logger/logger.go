package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// Init - 로거 초기화
func Init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = ""

	var err error
	Logger, err = config.Build(
		zap.AddCallerSkip(1), // 호출자 파일/라인 표시
	)
	if err != nil {
		panic(err)
	}
}

// Sync - 로거 플러시
func Sync() {
	_ = Logger.Sync()
}
