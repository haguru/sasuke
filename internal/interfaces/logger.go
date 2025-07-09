package interfaces

// Logger defines a generic logging interface.
type Logger interface {
	Info(msg string, keyvals ...interface{})
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
	Debug(msg string, keyvals ...interface{})
	SetLevel(level string)
	WithContext(ctx map[string]interface{}) Logger
}
