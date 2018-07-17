package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/uphy/redmine-sync/importer"
	"github.com/urfave/cli"
)

var version = "0.0.1"

func main() {
	app := cli.NewApp()
	app.Name = "redmine-sync"
	app.Version = version

	var endpoint string
	var apikey string

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "apikey",
			EnvVar:      "REDMINE_APIKEY",
			Destination: &apikey,
		},
		cli.StringFlag{
			Name:        "endpoint",
			EnvVar:      "REDMINE_ENDPOINT",
			Destination: &endpoint,
		},
	}

	app.Before = func(ctx *cli.Context) error {
		if !ctx.IsSet("apikey") {
			return errors.New("apikey is required")
		}
		if !ctx.IsSet("endpoint") {
			return errors.New("endpoint is required")
		}
		return nil
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name: "import",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "file,f",
				},
			},
			Action: func(ctx *cli.Context) error {
				var file string
				if !ctx.IsSet("file") {
					return errors.New("--file flag is requred")
				}
				file = ctx.String("file")
				i, err := importer.NewImporter(endpoint, apikey)
				if err != nil {
					return err
				}
				return i.Import(file)
			},
		},
		cli.Command{
			Name: "export",
			Action: func(ctx *cli.Context) error {
				i, err := importer.NewImporter(endpoint, apikey)
				if err != nil {
					return err
				}
				return i.Export(os.Stdout)
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
