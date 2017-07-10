package subcmd

import (
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/kamichidu/go-hariti"
	"github.com/urfave/cli"
)

var vimscriptTemplate *template.Template

func init() {
	const templateText = `" Generated by hariti {{.Version}}
" DO NOT EDIT THIS FILE
set runtimepath=
{{- range $path := $.RuntimePaths}}
set runtimepath+={{$path}}
{{- end}}
`
	vimscriptTemplate = template.Must(template.New("vim-script").
		Funcs(template.FuncMap{
			"pathjoin": func(elem ...string) string {
				return filepath.Join(elem...)
			},
		}).
		Parse(templateText),
	)
}

func deployAction(c *cli.Context) error {
	har := c.App.Metadata["hariti"].(*hariti.Hariti)

	var w io.Writer
	if ofile := c.String("output"); ofile == "-" {
		w = c.App.Writer
	} else {
		fw, err := os.Create(ofile)
		if err != nil {
			return cli.NewExitError(err, 128)
		}
		defer fw.Close()
		w = fw
	}

	rtp, err := har.RuntimeDirs()
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	data := map[string]interface{}{
		"Version":      c.App.Version,
		"RuntimePaths": rtp,
	}
	if err = vimscriptTemplate.Execute(w, data); err != nil {
		return cli.NewExitError(err, 1)
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
				Name:  "output,o",
				Usage: "Output `FILE`",
				Value: "-",
			},
		},
		Action: deployAction,
	})
}
