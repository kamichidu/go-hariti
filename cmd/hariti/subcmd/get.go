package subcmd

import (
	"context"
	"sync"

	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func getAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	wg := new(sync.WaitGroup)
	for _, arg := range c.Args() {
		wg.Add(1)
		go func(repository string) {
			defer wg.Done()

			logger := har.Logger.WithPrefix(repository)

			ctx := context.Background()
			ctx = hariti.WithLogger(ctx, logger)
			if err := har.Get(ctx, repository, c.Bool("update"), c.BoolT("disabled")); err != nil {
				logger.Error(err)
			}
		}(arg)
	}
	wg.Wait()

	return nil
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "get",
		Usage:     "Get bundle from {repository}",
		ArgsUsage: "{repository}...",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "u, update",
				Usage: "Update",
			},
			&cli.BoolTFlag{
				Name:  "disabled",
				Usage: "Only get, but disabled bundle",
			},
		},
		Action: getAction,
	})
}
