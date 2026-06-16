package logger

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/comail/colog"
	"github.com/kamichidu/go-hariti"
)

type adapter struct {
	cl *colog.CoLog
	l  *log.Logger
	mu *sync.Mutex
}

type LoggerOptions struct {
	Level hariti.LogLevel
}

func toCologLevel(lvl hariti.LogLevel) colog.Level {
	switch lvl {
	case hariti.LevelDebug:
		return colog.LDebug
	case hariti.LevelInfo:
		return colog.LInfo
	case hariti.LevelWarn:
		return colog.LWarning
	case hariti.LevelError:
		return colog.LError
	default:
		return colog.LInfo
	}
}

func NewLogger(w io.Writer, opts LoggerOptions) hariti.Logger {
	cl := colog.NewCoLog(w, "", 0)
	cl.SetMinLevel(toCologLevel(opts.Level))
	cl.SetDefaultLevel(colog.LInfo)
	return &adapter{
		cl: cl,
		l:  log.New(cl, "", 0),
		mu: new(sync.Mutex),
	}
}

func (l *adapter) Debug(args ...interface{}) {
	l.log(colog.LDebug, fmt.Sprint(args...))
}

func (l *adapter) Debugf(format string, args ...interface{}) {
	l.log(colog.LDebug, fmt.Sprintf(format, args...))
}

func (l *adapter) Info(args ...interface{}) {
	l.log(colog.LInfo, fmt.Sprint(args...))
}

func (l *adapter) Infof(format string, args ...interface{}) {
	l.log(colog.LInfo, fmt.Sprintf(format, args...))
}

func (l *adapter) Warn(args ...interface{}) {
	l.log(colog.LWarning, fmt.Sprint(args...))
}

func (l *adapter) Warnf(format string, args ...interface{}) {
	l.log(colog.LWarning, fmt.Sprintf(format, args...))
}

func (l *adapter) Error(args ...interface{}) {
	l.log(colog.LError, fmt.Sprint(args...))
}

func (l *adapter) Errorf(format string, args ...interface{}) {
	l.log(colog.LError, fmt.Sprintf(format, args...))
}

func (l *adapter) log(lvl colog.Level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var prefixStr string
	switch lvl {
	case colog.LTrace:
		prefixStr = "trace: "
	case colog.LDebug:
		prefixStr = "debug: "
	case colog.LInfo:
		prefixStr = "info: "
	case colog.LWarning:
		prefixStr = "warn: "
	case colog.LError:
		prefixStr = "error: "
	case colog.LAlert:
		prefixStr = "alert: "
	default:
		prefixStr = "info: "
	}

	lines := []string{}
	format := fmt.Sprintf("%s%%s", prefixStr)

	for _, line := range strings.Split(msg, "\n") {
		lines = append(lines, fmt.Sprintf(format, line))
	}
	msg = strings.Join(lines, "\n")

	l.l.Print(msg)
}
