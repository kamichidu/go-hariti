package subcmd

import (
	"log"

	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func enableAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)
	logger := c.App.Metadata["logger"].(*log.Logger)

	expr := c.String("when")
	for _, arg := range c.Args() {
		var err error
		if expr != "" {
			err = har.EnableIf(arg, expr)
		} else {
			err = har.Enable(arg)
		}
		if err != nil {
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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "when",
				Usage: "Enabled when given vim script evaluated as true",
			},
		},
		Action: enableAction,
	})
}
