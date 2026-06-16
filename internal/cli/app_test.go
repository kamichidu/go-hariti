package cli

import (
	"context"
	"flag"
	"testing"
)

type dummyCommand struct {
	name    string
	runFunc func(ctx *Context, args []string) error
}

func (c *dummyCommand) Name() string {
	return c.name
}

func (c *dummyCommand) Run(ctx *Context, args []string) error {
	fs := flag.NewFlagSet(c.name, flag.ContinueOnError)
	ctx.Global.Register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if c.runFunc != nil {
		return c.runFunc(ctx, fs.Args())
	}
	return nil
}

func TestRun_GlobalFlags(t *testing.T) {
	var lastCtx *Context
	var lastArgs []string

	cmd := &dummyCommand{
		name: "test-cmd",
		runFunc: func(ctx *Context, args []string) error {
			lastCtx = ctx
			lastArgs = args
			return nil
		},
	}
	Register(cmd)

	tests := []struct {
		name            string
		args            []string
		expectedCode    int
		expectConfig    string
		expectConfigDir string
		expectDataDir   string
		expectVerbose   bool
		expectArgs      []string
		expectNoExecute bool
	}{
		{
			name:         "global flags before subcommand",
			args:         []string{"-c", "/foo/bundles.hariti", "test-cmd"},
			expectedCode: 0,
			expectConfig: "/foo/bundles.hariti",
			expectArgs:   []string{},
		},
		{
			name:         "global flags after subcommand",
			args:         []string{"test-cmd", "-c", "/bar/bundles.hariti"},
			expectedCode: 0,
			expectConfig: "/bar/bundles.hariti",
			expectArgs:   []string{},
		},
		{
			name:            "global flags before and after with override precedence",
			args:            []string{"-config-dir", "A", "test-cmd", "-config-dir", "B"},
			expectedCode:    0,
			expectConfigDir: "B",
			expectArgs:      []string{},
		},
		{
			name:         "global flags after subcommand with positional argument",
			args:         []string{"test-cmd", "-c", "/baz/bundles.hariti", "extra-arg"},
			expectedCode: 0,
			expectConfig: "/baz/bundles.hariti",
			expectArgs:   []string{"extra-arg"},
		},
		{
			name:            "subcommand with help flag",
			args:            []string{"test-cmd", "--help"},
			expectedCode:    0,
			expectArgs:      []string{"--help"},
			expectNoExecute: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lastCtx = nil
			lastArgs = nil

			code := Run(context.Background(), tc.args)
			if code != tc.expectedCode {
				t.Fatalf("expected code %d, got %d", tc.expectedCode, code)
			}

			if tc.expectedCode == 0 {
				if tc.expectNoExecute {
					if lastCtx != nil {
						t.Fatal("expected dummy command NOT to be executed, but runFunc was called")
					}
					return
				}
				if lastCtx == nil {
					t.Fatal("expected dummy command to be executed, but runFunc was not called")
				}
				if tc.expectConfig != "" && lastCtx.Global.ConfigFile != tc.expectConfig {
					t.Errorf("expected ConfigFile %q, got %q", tc.expectConfig, lastCtx.Global.ConfigFile)
				}
				if tc.expectConfigDir != "" && lastCtx.Global.ConfigDir != tc.expectConfigDir {
					t.Errorf("expected ConfigDir %q, got %q", tc.expectConfigDir, lastCtx.Global.ConfigDir)
				}
				if tc.expectDataDir != "" && lastCtx.Global.DataDir != tc.expectDataDir {
					t.Errorf("expected DataDir %q, got %q", tc.expectDataDir, lastCtx.Global.DataDir)
				}
				if tc.expectVerbose && !lastCtx.Global.Verbose {
					t.Errorf("expected Verbose true, got false")
				}

				if len(lastArgs) != len(tc.expectArgs) {
					t.Fatalf("expected args len %d, got %d (%v)", len(tc.expectArgs), len(lastArgs), lastArgs)
				}
				for i, arg := range lastArgs {
					if arg != tc.expectArgs[i] {
						t.Errorf("expected arg[%d] %q, got %q", i, tc.expectArgs[i], arg)
					}
				}
			}
		})
	}
}
