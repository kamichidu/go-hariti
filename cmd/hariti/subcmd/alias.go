package subcmd

import (
	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func addAliasAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	repository := c.Args().First()
	for _, arg := range c.Args().Tail() {
		if err := har.AddAlias(repository, arg); err != nil {
			return cli.NewExitError(err, 1)
		}
	}

	return nil
}

func removeAliasAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	repository := c.Args().First()
	for _, arg := range c.Args().Tail() {
		if err := har.RemoveAlias(repository, arg); err != nil {
			return cli.NewExitError(err, 1)
		}
	}

	return nil
}

func clearAliasAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	return har.ClearAlias(c.Args().First())
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:  "alias",
		Usage: "Alias management",
		Flags: []cli.Flag{},
		Subcommands: cli.Commands{
			cli.Command{
				Name:      "add",
				Aliases:   []string{"a"},
				Usage:     "Add alias",
				ArgsUsage: "{repository} {alias}...",
				Action:    addAliasAction,
			},
			cli.Command{
				Name:      "rm",
				Aliases:   []string{"delete", "d"},
				Usage:     "Remove aliases list",
				ArgsUsage: "{repository} {alias}...",
				Action:    removeAliasAction,
			},
			cli.Command{
				Name:      "clear",
				Aliases:   []string{"c"},
				Usage:     "Clear aliases",
				ArgsUsage: "{repository}",
				Action:    clearAliasAction,
			},
		},
	})
}
