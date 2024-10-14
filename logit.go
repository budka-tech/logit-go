package logit

import (
	"fmt"
	"github.com/budka-tech/configo"
	"github.com/budka-tech/envo"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

func NewLogger(sen configo.Sentry, env *envo.Env, appVersion string) (*zap.Logger, error) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              fmt.Sprintf("https://%v@%v", sen.Key, sen.Host),
		TracesSampleRate: 1.0,
		Debug:            true,
		Environment:      env.String(),
	})

	if err != nil {
		panic("Ошибка инициализации Sentry: " + err.Error())
	}

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

	switch *env {
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

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	logger = logger.With(zap.String("version", appVersion))

	core := logger.Core()
	core = zapcore.NewTee(core, sentryCore(zap.ErrorLevel))
	logger = zap.New(core)

	return logger, nil
}

func sentryCore(minLevel zapcore.Level) zapcore.Core {
	return zapcore.RegisterHooks(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(zapcore.AddSync(sentryLogWriter{})),
		zap.NewAtomicLevelAt(minLevel),
	), func(entry zapcore.Entry) error {
		if entry.Level >= zapcore.ErrorLevel {
			sentry.CaptureMessage(entry.Message)
		}
		return nil
	})
}

type sentryLogWriter struct{}

func (s sentryLogWriter) Write(p []byte) (n int, err error) {
	sentry.CaptureMessage(string(p))
	return len(p), nil
}

func (s sentryLogWriter) Sync() error {
	sentry.Flush(2 * time.Second)
	return nil
}

func TraceId() string {
	return uuid.New().String()
}
