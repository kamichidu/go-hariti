package hariti

import (
	"context"
	"io"
	"log"
)

type Context struct {
	context.Context

	Writer    io.Writer
	ErrWriter io.Writer
	Logger    *log.Logger
}

func (self *Context) BoolFlag(name string) bool {
	v := self.Context.Value(name)
	if b, ok := v.(bool); ok {
		return b
	} else {
		return false
	}
}

func (self *Context) StringFlag(name string) string {
	v := self.Context.Value(name)
	if s, ok := v.(string); ok {
		return s
	} else {
		return ""
	}
}

func (self *Context) SetFlag(name string, value interface{}) {
	self.Context = context.WithValue(self.Context, name, value)
}
