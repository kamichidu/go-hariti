package subcmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

func listAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	bundles, err := har.List()
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	const (
		minwidth = 0
		tabwidth = 0
		padding  = 3
		padchar  = ' '
		flags    = 0x0
	)
	w := tabwriter.NewWriter(c.App.Writer, minwidth, tabwidth, padding, padchar, flags)
	defer w.Flush()
	lineFmt := "%v\t%v\t%v\t%v\t%v\t%v\t%v\n"
	fmt.Fprintf(w, lineFmt, "Kind", "Name", "Enabled", "EnabledIf", "URL/Path", "Aliases", "Dependencies")
	for _, bundle := range bundles {
		enabled := har.IsEnabled(bundle)
		switch v := bundle.(type) {
		case *hariti.RemoteBundle:
			dependencies := make([]string, 0)
			for _, dependency := range v.Dependencies {
				dependencies = append(dependencies, dependency.GetName())
			}
			fmt.Fprintf(w, lineFmt, "Remote", v.Name, enabled, v.EnableIfExpr, v.URL, v.Aliases, dependencies)
		case *hariti.LocalBundle:
			fmt.Fprintf(w, lineFmt, "Local", v.GetName(), enabled, "", v.LocalPath, []string{}, []string{})
		}
	}

	return nil
}

func init() {
	Commands = append(Commands, cli.Command{
		Name:      "list",
		Usage:     "Show managed repositories",
		ArgsUsage: " ",
		Flags:     []cli.Flag{},
		Action:    listAction,
	})
}
