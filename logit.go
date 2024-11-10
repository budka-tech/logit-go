package logit

import (
	"context"
	"fmt"
	"github.com/budka-tech/configo"
	"github.com/budka-tech/envo"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type logIt struct {
	logger *zap.Logger
}

type Logger interface {
	Debug(fields ...interface{})
	Info(ctx context.Context, message, op string, fields ...zap.Field)
	Warn(ctx context.Context, message, op string, fields ...zap.Field)
	Error(ctx context.Context, err error, op string, fields ...zap.Field)
	Fatal(ctx context.Context, err error, op string, fields ...zap.Field)
	NewTraceContext(traceId *string) context.Context
}

func MustNewLogger(appConf *configo.App, loggerConf *configo.Logger, senConf *configo.Sentry, env *envo.Env) Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:  "time",
		LevelKey: "level",
		NameKey:  "logger",
		//CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "traceId",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(loggerConf.TimeFormat),
		EncodeDuration: zapcore.StringDurationEncoder,
		//EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder

	if !env.IsLocal() {
		if senConf != nil {
			err := sentry.Init(sentry.ClientOptions{
				Dsn:              fmt.Sprintf("https://%v@%v", senConf.Key, senConf.Host),
				TracesSampleRate: 1.0,
				Debug:            true,
				Environment:      env.String(),
			})

			if err != nil {
				panic("Ошибка инициализации Sentry: " + err.Error())
			}
		}
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var cores []zapcore.Core

	if loggerConf.EnableConsole {
		consoleWriter := zapcore.Lock(os.Stdout)
		cores = append(cores, zapcore.NewCore(encoder, consoleWriter, zapcore.Level(loggerConf.ConsoleLevel)))
	}

	if loggerConf.EnableFile {
		rotationTime, err := time.ParseDuration(loggerConf.RotationTime)
		if err != nil {
			panic("Invalid rotation time: " + err.Error())
		}

		lumberjackLogger := &lumberjack.Logger{
			Filename:   filepath.Join(loggerConf.Dir, fileName(appConf.Name, appConf.Version)),
			MaxSize:    loggerConf.MaxSize, // megabytes
			MaxBackups: loggerConf.MaxBackups,
			MaxAge:     loggerConf.MaxAge, // days
			Compress:   loggerConf.Compress,
		}

		timeRotatingWriter := NewTimeRotatingWriter(lumberjackLogger, rotationTime)
		fileWriter := zapcore.AddSync(timeRotatingWriter)
		cores = append(cores, zapcore.NewCore(encoder, fileWriter, zapcore.Level(loggerConf.FileLevel)))
	}

	// Объединяем выводы
	core := zapcore.NewTee(cores...)

	// Создаем логгер
	logger := zap.New(core, zap.AddStacktrace(zapcore.ErrorLevel))

	// Добавляем дополнительные поля
	fields := []zap.Field{
		zap.String("appName", appConf.Name),
		zap.String("appVersion", appConf.Version),
	}

	logger = logger.With(fields...)

	return &logIt{logger: logger}
}

func NewNopLogger() Logger {
	nopCore := zapcore.NewNopCore()
	nopLogger := zap.New(nopCore)
	return &logIt{logger: nopLogger}
}

// Debug - логирование отладочной информации
func (receiver *logIt) Debug(fields ...interface{}) {
	fmt.Println(strings.Repeat("-", 80))

	debugTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[DEBUG] %s\n", debugTime)

	for i, field := range fields {
		switch v := field.(type) {
		case string:
			fmt.Printf("Поле %d: %s\n", i, v)
		case int, int32, int64:
			fmt.Printf("Поле %d: %d\n", i, v)
		case float32, float64:
			fmt.Printf("Поле %d: %f\n", i, v)
		case bool:
			fmt.Printf("Поле %d: %t\n", i, v)
		case error:
			fmt.Printf("Поле %d (ошибка): %s\n", i, v.Error())
		default:
			fmt.Printf("Поле %d: %+v\n", i, v)
		}
	}

	fmt.Println(strings.Repeat("-", 80))
}

// Info - логирование информационных сообщений
func (receiver *logIt) Info(ctx context.Context, message, op string, fields ...zap.Field) {
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Info(
		message,
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}, fields...)...,
	)
}

// Warn - логирование предупреждений
func (receiver *logIt) Warn(ctx context.Context, message, op string, fields ...zap.Field) {
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Warn(
		message,
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}, fields...)...,
	)
}

// Error - логирование ошибок
func (receiver *logIt) Error(ctx context.Context, err error, op string, fields ...zap.Field) {
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Error(
		err.Error(),
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}, fields...)...,
	)
	sentry.CaptureException(err)
}

// Fatal - логирование критических ошибок, завершает приложение
func (receiver *logIt) Fatal(ctx context.Context, err error, op string, fields ...zap.Field) {
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Fatal(
		err.Error(),
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}, fields...)...,
	)
	sentry.CaptureException(err)
}

func (receiver *logIt) TraceId(traceId *string) *string {
	if traceId != nil {
		return traceId
	}

	newTraceId := uuid.New().String()
	return &newTraceId
}

func (receiver *logIt) NewTraceContext(traceId *string) context.Context {
	if traceId == nil {
		newTraceId := uuid.New().String()
		traceId = &newTraceId
	}
	return context.WithValue(context.Background(), "traceId", *traceId)
}

func (receiver *logIt) getTraceIdFromContext(ctx context.Context) string {
	if traceId, ok := ctx.Value("traceId").(string); ok {
		return traceId
	}
	return uuid.New().String()
}

func fileName(appName, appVersion string) string {
	currentDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s_%s_%s.log", appName, appVersion, currentDate)
}
