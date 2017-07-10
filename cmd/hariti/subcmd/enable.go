package subcmd

import (
	"log"

	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func enableAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)
	logger := c.App.Metadata["logger"].(*log.Logger)

	for _, arg := range c.Args() {
		if err := har.Enable(arg); err != nil {
			logger.Printf("Error: %s", err)
		}
	}
	return nil
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
