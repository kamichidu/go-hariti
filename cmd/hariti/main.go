package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kamichidu/go-hariti"
	"github.com/kamichidu/go-hariti/cmd/hariti/subcmd"
	_ "github.com/kamichidu/go-hariti/vcs/git"
	"github.com/urfave/cli"
)

// XXX: Actual values are injected on build time
var (
	appVersion = "{{appVersion}}"
)

var defaultHaritiDirectory string

func init() {
	if runtime.GOOS == "windows" {
		defaultHaritiDirectory = filepath.Join(os.Getenv("AppData"), "hariti")
	} else {
		defaultHaritiDirectory = filepath.Join(os.Getenv("HOME"), ".hariti")
	}
}

func run() int {
	app := cli.NewApp()
	app.Name = "hariti"
	app.Version = appVersion
	// app.Description = ""
	// app.Usage = ""
	// app.UsageText = ""
	// app.ArgsUsage = ""
	app.Commands = subcmd.Commands
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:   "directory,d",
			Usage:  "`DIRECTORY` managed by hariti",
			EnvVar: "HARITI_HOME",
			Value:  defaultHaritiDirectory,
		},
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr

	app.Before = func(c *cli.Context) error {
		har := hariti.NewHariti(&hariti.HaritiConfig{
			Directory: c.String("directory"),
			Writer:    c.App.Writer,
			ErrWriter: c.App.ErrWriter,
		})
		if err := har.SetupEnv(); err != nil {
			return err
		}
		c.App.Metadata["hariti"] = har
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Printf("Something went wrong: %s", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
