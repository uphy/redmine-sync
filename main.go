package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-redmine"

	"strconv"

	"github.com/uphy/redmine-sync/sync"
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
				var in io.Reader
				var file string
				if ctx.IsSet("file") {
					file = ctx.String("file")
					f, err := os.Open(file)
					if err != nil {
						return err
					}
					defer f.Close()
					in = f
				} else {
					fmt.Fprintln(os.Stderr, "Reading from std-in")
					in = os.Stdin
				}

				s, err := sync.New(endpoint, apikey)
				if err != nil {
					return err
				}

				config, changed, err := s.Import(in)
				if err != nil {
					return err
				}
				if changed && file != "" {
					return config.SaveFile(file)
				}
				return nil
			},
		},
		cli.Command{
			Name: "export",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "project",
				},
				cli.StringFlag{
					Name: "status",
				},
			},
			Action: func(ctx *cli.Context) error {
				s, err := sync.New(endpoint, apikey)
				if err != nil {
					return err
				}
				filter := &redmine.IssueFilter{}
				if ctx.IsSet("project") {
					id, err := s.Projects.FindIDByName(ctx.String("project"))
					if err != nil {
						return err
					}
					filter.ProjectId = strconv.Itoa(id)
				}
				if ctx.IsSet("status") {
					id, err := s.Statuses.FindIDByName(ctx.String("status"))
					if err != nil {
						return err
					}
					filter.StatusId = strconv.Itoa(id)
				}
				config, err := s.Export(filter, os.Stdout)
				if err != nil {
					return err
				}
				return config.Save(os.Stdout)
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
