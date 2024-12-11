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
	Info(ctx context.Context, message string, fields ...zap.Field)
	Infof(ctx context.Context, message string, a ...any)
	Warn(ctx context.Context, message string, fields ...zap.Field)
	Warnf(ctx context.Context, message string, a ...any)
	Error(ctx context.Context, err error, fields ...zap.Field)
	Errorf(ctx context.Context, format string, args ...interface{})
	Fatal(ctx context.Context, err error, fields ...zap.Field)
	Fatalf(ctx context.Context, format string, args ...interface{})
	NewCtx(ctx context.Context, op string, traceId *string) context.Context
	NewOpCtx(ctx context.Context, op string) context.Context
	NewTraceCtx(ctx context.Context, traceId *string) context.Context
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

func (receiver *logIt) Info(ctx context.Context, message string, fields ...zap.Field) {
	op := receiver.getOpFromContext(ctx)
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Info(
		message,
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}, fields...)...,
	)
}

func (receiver *logIt) Infof(ctx context.Context, message string, a ...any) {
	op := receiver.getOpFromContext(ctx)
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Info(
		fmt.Sprintf(message, a),
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		})...,
	)
}

// Warn - логирование предупреждений
func (receiver *logIt) Warn(ctx context.Context, message string, fields ...zap.Field) {
	op := receiver.getOpFromContext(ctx)
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Warn(
		message,
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}, fields...)...,
	)
}

func (receiver *logIt) Warnf(ctx context.Context, message string, a ...any) {
	op := receiver.getOpFromContext(ctx)
	traceId := receiver.getTraceIdFromContext(ctx)
	receiver.logger.Warn(
		fmt.Sprintf(message, a),
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		})...,
	)
}

// Error - логирование ошибок
func (receiver *logIt) Error(ctx context.Context, err error, fields ...zap.Field) {
	op := receiver.getOpFromContext(ctx)
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

func (receiver *logIt) Errorf(ctx context.Context, format string, args ...interface{}) {
	op := receiver.getOpFromContext(ctx)
	traceId := receiver.getTraceIdFromContext(ctx)

	err := fmt.Errorf(format, args...)

	receiver.logger.Error(
		err.Error(),
		[]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}...,
	)

	sentry.CaptureException(err)
}

// Fatal - логирование критических ошибок, завершает приложение
func (receiver *logIt) Fatal(ctx context.Context, err error, fields ...zap.Field) {
	op := receiver.getOpFromContext(ctx)
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

func (receiver *logIt) Fatalf(ctx context.Context, format string, args ...interface{}) {
	op := receiver.getOpFromContext(ctx)
	traceId := receiver.getTraceIdFromContext(ctx)

	err := fmt.Errorf(format, args...)

	receiver.logger.Fatal(
		err.Error(),
		[]zap.Field{
			zap.String("op", op),
			zap.String("traceId", traceId),
		}...,
	)

	sentry.CaptureException(err)
}

// NewCtx создает новый контекст с указанной операцией и traceId
func (receiver *logIt) NewCtx(ctx context.Context, op string, traceId *string) context.Context {
	if traceId == nil {
		newTraceId := uuid.New().String()
		traceId = &newTraceId
	}
	ctx = context.WithValue(ctx, "op", op)
	return context.WithValue(ctx, "traceId", *traceId)
}

// NewOpCtx создает новый контекст с указанной операции
func (receiver *logIt) NewOpCtx(ctx context.Context, op string) context.Context {
	return context.WithValue(ctx, "op", op)
}

// NewTraceCtx создает новый контекст с указанным traceId
func (receiver *logIt) NewTraceCtx(ctx context.Context, traceId *string) context.Context {
	if traceId == nil {
		newTraceId := uuid.New().String()
		traceId = &newTraceId
	}
	return context.WithValue(ctx, "traceId", *traceId)
}

// getOpFromContext извлекает операцию из контекста
func (receiver *logIt) getOpFromContext(ctx context.Context) string {
	if op, ok := ctx.Value("op").(string); ok {
		return op
	}
	return "unknown"
}

// getTraceIdFromContext извлекает traceId из контекста
func (receiver *logIt) getTraceIdFromContext(ctx context.Context) string {
	if traceId, ok := ctx.Value("traceId").(string); ok {
		return traceId
	}
	return uuid.New().String()
}

func (receiver *logIt) NewTraceContext(traceId *string) context.Context {
	if traceId == nil {
		newTraceId := uuid.New().String()
		traceId = &newTraceId
	}
	return context.WithValue(context.Background(), "traceId", *traceId)
}

func fileName(appName, appVersion string) string {
	currentDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s_%s_%s.log", appName, appVersion, currentDate)
}
