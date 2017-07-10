package subcmd

import (
	"log"
	"sync"

	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func getAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)
	logger := c.App.Metadata["logger"].(*log.Logger)

	wg := new(sync.WaitGroup)
	for _, arg := range c.Args() {
		wg.Add(1)
		go func(repository string) {
			defer wg.Done()

			if err := har.Get(repository, c.Bool("update"), c.BoolT("disabled")); err != nil {
				logger.Printf("Error: %s", err)
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
