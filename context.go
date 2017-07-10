package hariti

import (
	"context"
	"io"
)

const (
	writerContextKey = iota
	errWriterContextKey
	loggerContextKey
)

func WithWriter(parent context.Context, w io.Writer) context.Context {
	return context.WithValue(parent, writerContextKey, w)
}

func WriterFromContext(ctx context.Context) io.Writer {
	return ctx.Value(writerContextKey).(io.Writer)
}

func WithErrWriter(parent context.Context, w io.Writer) context.Context {
	return context.WithValue(parent, errWriterContextKey, w)
}

func ErrWriterFromContext(ctx context.Context) io.Writer {
	return ctx.Value(errWriterContextKey).(io.Writer)
}

type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
}

func WithLogger(parent context.Context, logger Logger) context.Context {
	return context.WithValue(parent, loggerContextKey, logger)
}

func LoggerFromContextKey(ctx context.Context) Logger {
	return ctx.Value(loggerContextKey).(Logger)
}
