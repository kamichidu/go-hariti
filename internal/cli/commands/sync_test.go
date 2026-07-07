package commands

import (
	"context"
	"flag"
	"testing"

	"github.com/kamichidu/go-flagshim"
)

func TestSyncCommand_Flags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedVal int
	}{
		{
			name:        "no flag",
			args:        []string{},
			expectedVal: 0,
		},
		{
			name:        "short flag",
			args:        []string{"-p", "4"},
			expectedVal: 4,
		},
		{
			name:        "long flag",
			args:        []string{"--parallelism", "16"},
			expectedVal: 16,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &SyncCommand{}
			fs := flagshim.NewFlagSet(cmd.Name(), flag.ContinueOnError)
			_ = cmd.RegisterFlags(context.Background(), fs)

			err := fs.Parse(tc.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			if cmd.parallelism != tc.expectedVal {
				t.Errorf("expected parallelism to be %d, got %d", tc.expectedVal, cmd.parallelism)
			}
		})
	}
}
