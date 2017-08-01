package hariti

import (
	"fmt"
	"io"
	"log"
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
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Panic(...interface{})
	Panicf(string, ...interface{})
}

type stdLogger struct {
	l *log.Logger
}

func NewStdLogger(w io.Writer) Logger {
	return &stdLogger{log.New(w, "", 0x0)}
}

func (self *stdLogger) Debug(args ...interface{}) {
	self.l.Printf("> D | %s", fmt.Sprint(args...))
}

func (self *stdLogger) Debugf(format string, args ...interface{}) {
	self.Debug(fmt.Sprintf(format, args...))
}

func (self *stdLogger) Info(args ...interface{}) {
	self.l.Printf("> I | %s", fmt.Sprint(args...))
}

func (self *stdLogger) Infof(format string, args ...interface{}) {
	self.Info(fmt.Sprintf(format, args...))
}

func (self *stdLogger) Warn(args ...interface{}) {
	self.l.Printf("> W | %s", fmt.Sprint(args...))
}

func (self *stdLogger) Warnf(format string, args ...interface{}) {
	self.Warn(fmt.Sprintf(format, args...))
}

func (self *stdLogger) Error(args ...interface{}) {
	self.l.Printf("> E | %s", fmt.Sprint(args...))
}

func (self *stdLogger) Errorf(format string, args ...interface{}) {
	self.Error(fmt.Sprintf(format, args...))
}

func (self *stdLogger) Fatal(args ...interface{}) {
	self.l.Fatalf("> F | %s", fmt.Sprint(args...))
}

func (self *stdLogger) Fatalf(format string, args ...interface{}) {
	self.Fatal(fmt.Sprintf(format, args...))
}

func (self *stdLogger) Panic(args ...interface{}) {
	self.l.Panicf("> P | %s", fmt.Sprint(args...))
}

func (self *stdLogger) Panicf(format string, args ...interface{}) {
	self.Panic(fmt.Sprintf(format, args...))
}
