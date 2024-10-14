package logit_go

import (
	"github.com/budka-tech/envo"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(env envo.Env) (*zap.Logger, error) {
	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
		Development: true,
		Encoding:    "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "op",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	switch env {
	case "prod":
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		cfg.Development = false
		cfg.Sampling = &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
	default:
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.NameKey = "op"
	}

	return cfg.Build()
}

func TraceId() string {
	return uuid.New().String()
}
