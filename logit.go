package logit

import (
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
	"time"
)

type LogIt struct {
	logger *zap.Logger
}

// Debug - логирование отладочной информации
func (receiver *LogIt) Debug(op string, traceId *string, fields ...zap.Field) {
	traceId = receiver.TraceId(traceId)
	receiver.logger.Debug(
		"Debug",
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", *traceId),
		}, fields...)...,
	)
}

// Info - логирование информационных сообщений
func (receiver *LogIt) Info(message, op string, traceId *string, fields ...zap.Field) {
	traceId = receiver.TraceId(traceId)
	receiver.logger.Info(
		message,
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", *traceId),
		}, fields...)...,
	)
}

// Warn - логирование предупреждений
func (receiver *LogIt) Warn(message, op string, traceId *string, fields ...zap.Field) {
	traceId = receiver.TraceId(traceId)
	receiver.logger.Warn(
		message,
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", *traceId),
		}, fields...)...,
	)
}

// Error - логирование ошибок
func (receiver *LogIt) Error(err error, op string, traceId *string, fields ...zap.Field) {
	traceId = receiver.TraceId(traceId)
	receiver.logger.Error(
		err.Error(),
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", *traceId),
		}, fields...)...,
	)
	sentry.CaptureException(err)
}

// Fatal - логирование критических ошибок, завершает приложение
func (receiver *LogIt) Fatal(err error, op string, traceId *string, fields ...zap.Field) {
	traceId = receiver.TraceId(traceId)
	receiver.logger.Fatal(
		err.Error(),
		append([]zap.Field{
			zap.String("op", op),
			zap.String("traceId", *traceId),
		}, fields...)...,
	)
	sentry.CaptureException(err)
}

// MustNewLogger - инициализация нового логгера
func MustNewLogger(appConf *configo.App, loggerConf *configo.Logger, senConf *configo.Sentry, env *envo.Env) *LogIt {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "traceId",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(loggerConf.TimeFormat),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
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
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Добавляем дополнительные поля
	fields := []zap.Field{
		zap.String("appName", appConf.Name),
		zap.String("appVersion", appConf.Version),
	}

	logger = logger.With(fields...)

	return &LogIt{logger: logger}
}

// TraceId - генерация уникального идентификатора трассировки
func (receiver *LogIt) TraceId(traceId *string) *string {
	if traceId != nil {
		return traceId
	}

	newTraceId := uuid.New().String()
	return &newTraceId
}

func fileName(appName, appVersion string) string {
	currentDate := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s_%s_%s.log", appName, appVersion, currentDate)
}
