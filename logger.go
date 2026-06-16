package hariti

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/kamichidu/go-hariti/vcs"
)

type Logger interface {
	WithPrefix(string) Logger

	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Panic(...interface{})
	Panicf(string, ...interface{})
}

type stdLogger struct {
	l *log.Logger

	w io.Writer

	prefix string

	mu *sync.Mutex
}

func NewStdLogger(w io.Writer) Logger {
	return &stdLogger{log.New(w, "", 0x0), w, "", new(sync.Mutex)}
}

func (l *stdLogger) WithPrefix(prefix string) Logger {
	return &stdLogger{log.New(l.w, "", l.l.Flags()), l.w, prefix, l.mu}
}

func (l *stdLogger) Debug(args ...interface{}) {
	l.log('D', fmt.Sprint(args...))
}

func (l *stdLogger) Debugf(format string, args ...interface{}) {
	l.log('D', fmt.Sprintf(format, args...))
}

func (l *stdLogger) Info(args ...interface{}) {
	l.log('I', fmt.Sprint(args...))
}

func (l *stdLogger) Infof(format string, args ...interface{}) {
	l.log('I', fmt.Sprintf(format, args...))
}

func (l *stdLogger) Warn(args ...interface{}) {
	l.log('W', fmt.Sprint(args...))
}

func (l *stdLogger) Warnf(format string, args ...interface{}) {
	l.log('W', fmt.Sprintf(format, args...))
}

func (l *stdLogger) Error(args ...interface{}) {
	l.log('E', fmt.Sprint(args...))
}

func (l *stdLogger) Errorf(format string, args ...interface{}) {
	l.log('E', fmt.Sprintf(format, args...))
}

func (l *stdLogger) Fatal(args ...interface{}) {
	l.log('F', fmt.Sprint(args...))
}

func (l *stdLogger) Fatalf(format string, args ...interface{}) {
	l.log('F', fmt.Sprintf(format, args...))
}

func (l *stdLogger) Panic(args ...interface{}) {
	l.log('P', fmt.Sprint(args...))
}

func (l *stdLogger) Panicf(format string, args ...interface{}) {
	l.log('P', fmt.Sprintf(format, args...))
}

func (l *stdLogger) log(lvl rune, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	lines := []string{}
	format := fmt.Sprintf("> %c | %-30s | %%s", lvl, l.prefix)
	for i, line := range strings.Split(msg, "\n") {
		if i == 1 {
			format = strings.Repeat(" ", len(format)-len(" | %s")) + " | %s"
		}
		lines = append(lines, fmt.Sprintf(format, line))
	}
	msg = strings.Join(lines, "\n")

	switch lvl {
	case 'F':
		l.l.Fatal(msg)
	case 'P':
		l.l.Panic(msg)
	default:
		l.l.Print(msg)
	}
}

func WithLogger(parent context.Context, logger Logger) context.Context {
	return vcs.WithLogger(parent, logger)
}

func LoggerFromContextKey(ctx context.Context) Logger {
	if val := vcs.LoggerFromContext(ctx); val != nil {
		if l, ok := val.(Logger); ok {
			return l
		}
	}
	return NewStdLogger(io.Discard)
}
