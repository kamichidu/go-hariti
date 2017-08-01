package hariti

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
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

func (self *stdLogger) WithPrefix(prefix string) Logger {
	return &stdLogger{log.New(self.w, "", self.l.Flags()), self.w, prefix, self.mu}
}

func (self *stdLogger) Debug(args ...interface{}) {
	self.log('D', fmt.Sprint(args...))
}

func (self *stdLogger) Debugf(format string, args ...interface{}) {
	self.log('D', fmt.Sprintf(format, args...))
}

func (self *stdLogger) Info(args ...interface{}) {
	self.log('I', fmt.Sprint(args...))
}

func (self *stdLogger) Infof(format string, args ...interface{}) {
	self.log('I', fmt.Sprintf(format, args...))
}

func (self *stdLogger) Warn(args ...interface{}) {
	self.log('W', fmt.Sprint(args...))
}

func (self *stdLogger) Warnf(format string, args ...interface{}) {
	self.log('W', fmt.Sprintf(format, args...))
}

func (self *stdLogger) Error(args ...interface{}) {
	self.log('E', fmt.Sprint(args...))
}

func (self *stdLogger) Errorf(format string, args ...interface{}) {
	self.log('E', fmt.Sprintf(format, args...))
}

func (self *stdLogger) Fatal(args ...interface{}) {
	self.log('F', fmt.Sprint(args...))
}

func (self *stdLogger) Fatalf(format string, args ...interface{}) {
	self.log('F', fmt.Sprintf(format, args...))
}

func (self *stdLogger) Panic(args ...interface{}) {
	self.log('P', fmt.Sprint(args...))
}

func (self *stdLogger) Panicf(format string, args ...interface{}) {
	self.log('P', fmt.Sprintf(format, args...))
}

func (self *stdLogger) log(lvl rune, msg string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	lines := []string{}
	format := fmt.Sprintf("> %c | %-30s | %%s", lvl, self.prefix)
	for i, line := range strings.Split(msg, "\n") {
		if i == 1 {
			format = strings.Repeat(" ", len(format)-len(" | %s")) + " | %s"
		}
		lines = append(lines, fmt.Sprintf(format, line))
	}
	msg = strings.Join(lines, "\n")

	switch lvl {
	case 'F':
		self.l.Fatal(msg)
	case 'P':
		self.l.Panic(msg)
	default:
		self.l.Print(msg)
	}
}
