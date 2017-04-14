package subcmd

import (
	"fmt"
	"io"
	"os"

	"github.com/kamichidu/go-hariti/encoding/hariti"
	"github.com/urfave/cli"
)

func deployAction(c *cli.Context) error {
	src, err := os.Open(c.Args().First())
	if err != nil {
		return err
	}

	bundles, err := hariti.Parse(src)
	if err != nil {
		return err
	}
	for i, bundle := range bundles {
		io.WriteString(c.App.Writer, fmt.Sprintf("%2d: %#v\n", i, bundle))
	}
	return nil
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "deploy",
		Usage:     "Generate vim script for setting-up runtimepath",
		ArgsUsage: "{bundles file}",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "o",
				Usage: "Output `filename`",
				Value: "-",
			},
		},
		Action: deployAction,
	})
}
