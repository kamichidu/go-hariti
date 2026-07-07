package cli

import (
	"context"
	"io"

	"github.com/kamichidu/go-flagshim"
	"github.com/kamichidu/go-hariti"
)

type GlobalFlags struct {
	ConfigFile string
	ConfigDir  string
	DataDir    string
	Verbose    bool
}

func (g *GlobalFlags) Register(ctx context.Context, fs *flagshim.FlagSet) {
	fs.StringVar(&g.ConfigFile, "config", g.ConfigFile, "")
	fs.Alias("config", "c")
	fs.StringVar(&g.ConfigDir, "config-dir", g.ConfigDir, "")
	fs.StringVar(&g.DataDir, "data-dir", g.DataDir, "")
	fs.BoolVar(&g.Verbose, "verbose", g.Verbose, "")
	fs.Alias("verbose", "v")
}

type contextKeyLogger struct{}

func ContextWithLogger(ctx context.Context, logger hariti.Logger) context.Context {
	return context.WithValue(ctx, contextKeyLogger{}, logger)
}

func LoggerFromContext(ctx context.Context) hariti.Logger {
	if logger, ok := ctx.Value(contextKeyLogger{}).(hariti.Logger); ok {
		return logger
	}
	return nil
}

func GetGlobalFlags(ctx context.Context) *GlobalFlags {
	return flagshim.MustFlagFromContext[GlobalFlags](ctx)
}

func GetLogger(ctx context.Context) hariti.Logger {
	return LoggerFromContext(ctx)
}

func GetStdout(ctx context.Context) io.Writer {
	return flagshim.MustStdoutFromContext(ctx)
}

func GetStderr(ctx context.Context) io.Writer {
	return flagshim.MustStderrFromContext(ctx)
}
