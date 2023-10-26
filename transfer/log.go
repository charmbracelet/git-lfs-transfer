package transfer

// Logger is a logging interface.
type Logger interface {
	Log(v ...interface{})
	Logf(format string, v ...interface{})
}

type noopLogger struct{}

var _ Logger = (*noopLogger)(nil)

// Log implements Logger.
func (*noopLogger) Log(v ...interface{}) {}

// Logf implements Logger.
func (*noopLogger) Logf(format string, v ...interface{}) {}
