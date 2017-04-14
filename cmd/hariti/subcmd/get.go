package subcmd

import (
	"fmt"
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

			if err := har.Get(repository); err != nil {
				fmt.Printf("Error: %s\n", err)
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
		Flags:     []cli.Flag{},
		Action:    getAction,
	})
}
