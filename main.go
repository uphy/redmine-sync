package main

import (
	"errors"
	"fmt"
	"os"

	redmine "github.com/uphy/go-redmine"

	"strconv"

	"github.com/uphy/redmine-sync/sync"
	"github.com/urfave/cli"
)

var version = "0.0.2"

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
			Name: "watch",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "file,f",
				},
			},
			ArgsUsage: "[file]",
			Action: func(ctx *cli.Context) error {
				if ctx.NArg() != 1 {
					return errors.New("specify a file to watch")
				}
				s, err := sync.New(endpoint, apikey)
				if err != nil {
					return err
				}
				return s.Watch(ctx.Args().First(), true)
			},
		},
		cli.Command{
			Name: "import",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name: "base,b",
				},
			},
			ArgsUsage: "[file]",
			Action: func(ctx *cli.Context) error {
				var file string
				if ctx.NArg() != 1 {
					return errors.New("specify a file to import")
				}
				file = ctx.Args().First()
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()

				var base *os.File
				if ctx.IsSet("base") {
					f, err := os.Open(ctx.String("base"))
					if err != nil {
						return err
					}
					defer f.Close()
					base = f
				}

				s, err := sync.New(endpoint, apikey)
				if err != nil {
					return err
				}

				config, changed, err := s.ImportFile(f, base)
				if err != nil {
					return err
				}
				if changed && file != "" {
					f, err := os.Create(file)
					if err != nil {
						return err
					}
					defer f.Close()
					return s.Converter.SaveConfig(f, config)
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
				cli.StringFlag{
					Name:  "format",
					Value: "yaml",
				},
			},
			Action: func(ctx *cli.Context) error {
				s, err := sync.New(endpoint, apikey)
				if err != nil {
					return err
				}
				filter := &redmine.IssueFilter{}
				if ctx.IsSet("project") {
					id, err := s.Converter.Projects.FindIDByName(ctx.String("project"))
					if err != nil {
						return err
					}
					filter.ProjectId = strconv.Itoa(id)
				}
				if ctx.IsSet("status") {
					id, err := s.Converter.Statuses.FindIDByName(ctx.String("status"))
					if err != nil {
						return err
					}
					filter.StatusId = strconv.Itoa(id)
				}
				config, err := s.Export(filter, os.Stdout)
				if err != nil {
					return err
				}
				if ctx.IsSet("format") {
					format := ctx.String("format")
					switch format {
					case "yaml":
						return s.Converter.SaveConfigYAML(os.Stdout, config)
					case "csv":
						return s.Converter.SaveConfigCSV(os.Stdout, config)
					default:
						return errors.New("unsupported format: " + format)
					}
				} else {
					return s.Converter.SaveConfigCSV(os.Stdout, config)
				}
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
