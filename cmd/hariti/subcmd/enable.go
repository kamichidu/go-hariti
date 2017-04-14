package subcmd

import (
	"fmt"

	"github.com/urfave/cli"
)

func enableAction(c *cli.Context) error {
	return fmt.Errorf("Sorry, unimplemented yet")
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "enable",
		Usage:     "Enable {repository}",
		ArgsUsage: "{repository}...",
		Flags:     []cli.Flag{},
		Action:    enableAction,
	})
}
