package subcmd

import (
	"fmt"

	"github.com/urfave/cli"
)

func inspectAction(c *cli.Context) error {
	return fmt.Errorf("Sorry, unimplemented yet")
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "inspect",
		Usage:     "Show repository information",
		ArgsUsage: "{repository}",
		Flags:     []cli.Flag{},
		Action:    inspectAction,
	})
}
