# LogIt-Go

LogIt-Go - это мощная и гибкая библиотека для логирования в Go-приложениях, построенная на основе [Zap](https://github.com/uber-go/zap) и интегрированная с [Sentry](https://sentry.io/).

## Особенности

- Высокопроизводительное логирование с использованием Zap
- Поддержка ротации логов
- Интеграция с Sentry для отслеживания ошибок
- Настраиваемые уровни логирования для консоли и файла
- Автоматическое генерирование ID трассировки
- Поддержка структурированного логирования

## Установка

```bash
go get github.com/budka-tech/logit-go


## Использование
### Инициализация
```go
import (
    "github.com/budka-tech/logit-go"
    "github.com/budka-tech/configo"
    "github.com/budka-tech/envo"
)

func main() {
    appConf := &configo.App{...}
    loggerConf := &configo.Logger{...}
    senConf := &configo.Sentry{...}
    env := envo.New()

    logger := logit.MustNewLogger(appConf, loggerConf, senConf, env)

    // Использование логгера
    logger.Info("Application started", "main", nil)
}
```

### Логирование
```go
logger.Debug("Debug message", "operation", nil)
logger.Info("Info message", "operation", nil)
logger.Warn("Warning message", "operation", nil)
logger.Error(err, "operation", nil)
logger.Fatal(err, "operation", nil)
```

### Тестирование

Для использования в тестах предусмотрен пустой логгер

```go
logger := logit.NewNopLogger()
```

### Конфигурация логгера

Логгер принимает следующие параметры конфигурации:

#### App конфигурация (configo.App):
- `Name`: Имя приложения
- `Version`: Версия приложения

#### Logger конфигурация (configo.Logger):
- `EnableConsole`: Включить вывод в консоль (bool)
- `ConsoleLevel`: Уровень логирования для консоли (int)
- `EnableFile`: Включить запись в файл (bool)
- `FileLevel`: Уровень логирования для файла (int)
- `Dir`: Директория для хранения лог-файлов
- `MaxSize`: Максимальный размер файла лога в мегабайтах
- `MaxBackups`: Максимальное количество старых лог-файлов для хранения
- `MaxAge`: Максимальное время хранения старых лог-файлов в днях
- `Compress`: Сжимать ротированные лог-файлы (bool)
- `TimeFormat`: Формат времени для логов
- `RotationTime`: Интервал ротации логов (например, "24h")

#### Sentry конфигурация (configo.Sentry):
- `Key`: Ключ проекта Sentry
- `Host`: Хост Sentry

#### Env конфигурация (envo.Env):
- Объект, представляющий текущее окружение

Пример конфигурации:
##### Go
```go
appConf := &configo.App{
    Name:    "MyApp",
    Version: "1.0.0",
}

loggerConf := &configo.Logger{
    EnableConsole: true,
    ConsoleLevel:  int(zapcore.InfoLevel),
    EnableFile:    true,
    FileLevel:     int(zapcore.DebugLevel),
    Dir:           "/var/log/myapp",
    MaxSize:       100,
    MaxBackups:    3,
    MaxAge:        28,
    Compress:      true,
    TimeFormat:    "2006-01-02 15:04:05",
    RotationTime:  "24h",
}

senConf := &configo.Sentry{
    Key:  "your-sentry-key",
    Host: "sentry.io",
}

env := envo.New()

logger := logit.MustNewLogger(appConf, loggerConf, senConf, env)
```

Yaml

```yaml
app:
  name: "MyApp"
  version: "1.0.0"

logger:
  level: 0
  dir: "logs"
  maxSize: 10
  maxBackups: 3
  maxAge: 365
  compress: true
  rotationTime: "24h"
  consoleLevel: 0
  fileLevel: 0
  enableConsole: true
  enableFile: true
  timeFormat: "2006-01-02T15:04:05.000Z07:00"
```


### Описание полей
#### App конфигурация

| Параметр | Описание |
|----------|----------|
| `name`   | Имя приложения |
| `version`| Версия приложения |

#### Logger конфигурация

| Параметр | Описание | Тип | Значение по умолчанию |
|----------|----------|-----|------------------------|
| `level` | Общий уровень логирования | int | 0 |
| `dir` | Директория для хранения лог-файлов | string | "logs" |
| `maxSize` | Максимальный размер файла лога в мегабайтах | int | 10 |
| `maxBackups` | Максимальное количество старых лог-файлов для хранения | int | 3 |
| `maxAge` | Максимальное время хранения старых лог-файлов в днях | int | 365 |
| `compress` | Сжимать ротированные лог-файлы | bool | true |
| `rotationTime` | Интервал ротации логов | string | "24h" |
| `consoleLevel` | Уровень логирования для консоли | int | 0 |
| `fileLevel` | Уровень логирования для файла | int | 0 |
| `enableConsole` | Включить вывод в консоль | bool | true |
| `enableFile` | Включить запись в файл | bool | true |
| `timeFormat` | Формат времени для логов | string | "2006-01-02T15:04:05.000Z07:00" |