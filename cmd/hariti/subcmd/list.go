package subcmd

import (
	"fmt"

	"github.com/urfave/cli"
)

func listAction(c *cli.Context) error {
	return fmt.Errorf("Sorry, unimplemented yet")
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "list",
		Usage:     "Show managed repositories",
		ArgsUsage: " ",
		Flags:     []cli.Flag{},
		Action:    listAction,
	})
}
