package vcs

import (
	"context"
	"io"
	"os"
)

type contextKey string

const (
	writerContextKey    contextKey = "vcs:writer"
	errWriterContextKey contextKey = "vcs:errwriter"
	loggerContextKey    contextKey = "vcs:logger"
)

type Logger interface {
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type nopLogger struct{}

func (nopLogger) Infof(format string, args ...interface{})  {}
func (nopLogger) Debugf(format string, args ...interface{}) {}

func WithWriter(parent context.Context, w io.Writer) context.Context {
	return context.WithValue(parent, writerContextKey, w)
}

func WriterFromContext(ctx context.Context) io.Writer {
	if val := ctx.Value(writerContextKey); val != nil {
		if w, ok := val.(io.Writer); ok {
			return w
		}
	}
	return os.Stdout
}

func WithErrWriter(parent context.Context, w io.Writer) context.Context {
	return context.WithValue(parent, errWriterContextKey, w)
}

func ErrWriterFromContext(ctx context.Context) io.Writer {
	if val := ctx.Value(errWriterContextKey); val != nil {
		if w, ok := val.(io.Writer); ok {
			return w
		}
	}
	return os.Stderr
}

func WithLogger(parent context.Context, logger Logger) context.Context {
	return context.WithValue(parent, loggerContextKey, logger)
}

func LoggerFromContext(ctx context.Context) Logger {
	if val := ctx.Value(loggerContextKey); val != nil {
		if l, ok := val.(Logger); ok {
			return l
		}
	}
	return nopLogger{}
}
