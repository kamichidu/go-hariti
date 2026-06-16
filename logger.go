package hariti

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
}

type nopLogger struct{}

func (nopLogger) Debug(...interface{})          {}
func (nopLogger) Debugf(string, ...interface{}) {}
func (nopLogger) Info(...interface{})           {}
func (nopLogger) Infof(string, ...interface{})  {}
func (nopLogger) Warn(...interface{})           {}
func (nopLogger) Warnf(string, ...interface{})  {}
func (nopLogger) Error(...interface{})          {}
func (nopLogger) Errorf(string, ...interface{}) {}
