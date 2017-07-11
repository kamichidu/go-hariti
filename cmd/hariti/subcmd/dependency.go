package subcmd

import (
	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func addDependencyAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	repository := c.Args().First()
	for _, arg := range c.Args().Tail() {
		if err := har.AddDependency(repository, arg); err != nil {
			return cli.NewExitError(err, 1)
		}
	}

	return nil
}

func removeDependencyAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	repository := c.Args().First()
	for _, arg := range c.Args().Tail() {
		if err := har.RemoveDependency(repository, arg); err != nil {
			return cli.NewExitError(err, 1)
		}
	}

	return nil
}

func clearDependencyAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	return har.ClearDependencies(c.Args().First())
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:    "dependency",
		Aliases: []string{"dep"},
		Usage:   "Dependency management",
		Flags:   []cli.Flag{},
		Subcommands: cli.Commands{
			cli.Command{
				Name:      "add",
				Aliases:   []string{"a"},
				Usage:     "Add dependencies",
				ArgsUsage: "{repository} {dependency}...",
				Action:    addDependencyAction,
			},
			cli.Command{
				Name:      "rm",
				Aliases:   []string{"delete", "d"},
				Usage:     "Remove dependencies list",
				ArgsUsage: "{repository} {dependency}...",
				Action:    removeDependencyAction,
			},
			cli.Command{
				Name:      "clear",
				Aliases:   []string{"c"},
				Usage:     "Clear dependencies",
				ArgsUsage: "{repository}",
				Action:    clearDependencyAction,
			},
		},
	})
}
