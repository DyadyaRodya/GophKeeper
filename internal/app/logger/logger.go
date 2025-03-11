package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	cfg.EncoderConfig.CallerKey = zapcore.OmitKey
	Log, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return Log, nil
}
