package subcmd

import (
	"log"

	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func disableAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)
	logger := c.App.Metadata["logger"].(*log.Logger)

	for _, arg := range c.Args() {
		if err := har.Disable(arg); err != nil {
			logger.Printf("Error: %s", err)
		}
	}
	return nil
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
