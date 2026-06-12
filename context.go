package hariti

import (
	"context"
	"io"
)

type contextKey int

const (
	writerContextKey contextKey = iota
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

func WithLogger(parent context.Context, logger Logger) context.Context {
	return context.WithValue(parent, loggerContextKey, logger)
}

func LoggerFromContextKey(ctx context.Context) Logger {
	if val := ctx.Value(loggerContextKey); val != nil {
		if l, ok := val.(Logger); ok {
			return l
		}
	}
	return NewStdLogger(io.Discard)
}
