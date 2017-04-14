package subcmd

import (
	"fmt"

	"github.com/urfave/cli"
)

func disableAction(c *cli.Context) error {
	return fmt.Errorf("Sorry, unimplemented yet")
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "disable",
		Usage:     "Disable {repository}",
		ArgsUsage: "{repository}...",
		Flags:     []cli.Flag{},
		Action:    disableAction,
	})
}
