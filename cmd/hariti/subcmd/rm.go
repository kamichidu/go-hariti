package subcmd

import (
	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func rmAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	for _, arg := range c.Args() {
		if err := har.Remove(arg, c.Bool("force")); err != nil {
			return cli.NewExitError(err, 1)
		}
	}
	return nil
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "rm",
		Usage:     "Remove {repository}",
		ArgsUsage: "{repository}...",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "force,f",
				Usage: "Force remove",
			},
		},
		Action: rmAction,
	})
}
