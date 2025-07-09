package zerolog

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/rs/zerolog"
)

// Logger implements LoggerInterface using zerolog.
type Logger struct {
	zlog zerolog.Logger
}

// NewZerologLogger initializes zerolog with standard settings.
func NewZerologLogger(serviceName string) interfaces.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i any) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
    
	z := zerolog.New(output).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()
	return &Logger{zlog: z}
}

func (l *Logger) Info(msg string, keyvals ...interface{}) {
	event := l.zlog.Info()
	for i := 0; i < len(keyvals)-1; i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keyvals[i+1])
	}
	event.Msg(msg)
}

func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	event := l.zlog.Warn()
	for i := 0; i < len(keyvals)-1; i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keyvals[i+1])
	}
	event.Msg(msg)
}

func (l *Logger) Error(msg string, keyvals ...interface{}) {
	event := l.zlog.Error()
	for i := 0; i < len(keyvals)-1; i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keyvals[i+1])
	}
	event.Msg(msg)
}

func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	event := l.zlog.Debug()
	for i := 0; i < len(keyvals)-1; i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, keyvals[i+1])
	}
	event.Msg(msg)
}

// SetLevel sets the global log level for zerolog.
func (l *Logger) SetLevel(level string) {
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// WithContext creates a new logger with additional context.
func (l *Logger) WithContext(ctx map[string]interface{}) interfaces.Logger {
	newLogger := l.zlog.With()
	for key, value := range ctx {
		newLogger = newLogger.Interface(key, value)
	}
	return &Logger{zlog: newLogger.Logger()}
}
