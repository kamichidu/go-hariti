package subcmd

import (
	"fmt"

	"github.com/urfave/cli"
)

func rmAction(c *cli.Context) error {
	return fmt.Errorf("Sorry, unimplemented yet")
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "rm",
		Usage:     "Remove {repository}",
		ArgsUsage: "{repository}...",
		Flags:     []cli.Flag{},
		Action:    rmAction,
	})
}
