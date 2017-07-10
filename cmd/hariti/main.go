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

var appVersion string

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
		&cli.BoolFlag{
			Name:  "verbose",
			Usage: "Verbose output",
		},
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr

	app.Before = func(c *cli.Context) error {
		har := hariti.NewHariti(&hariti.HaritiConfig{
			Directory: c.String("directory"),
			Writer:    c.App.Writer,
			ErrWriter: c.App.ErrWriter,
			Verbose:   c.GlobalBool("verbose"),
		})
		if err := har.SetupManagedDirectory(); err != nil {
			return err
		}
		c.App.Metadata["hariti"] = har
		c.App.Metadata["logger"] = log.New(c.App.ErrWriter, "", 0x0)
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
