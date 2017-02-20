package main

import (
	"fmt"

	"github.com/cncd/pipeline/pipeline/frontend/yaml"
	"github.com/cncd/pipeline/pipeline/frontend/yaml/linter"

	"github.com/urfave/cli"
)

var lintCommand = cli.Command{
	Name:   "lint",
	Usage:  "lints the yaml file",
	Action: lintAction,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name: "trusted",
		},
	},
}

func lintAction(c *cli.Context) error {
	file := c.Args().First()
	if file == "" {
		return fmt.Errorf("Error: please provide a path the configuration file")
	}

	conf, err := yaml.ParseFile(file)
	if err != nil {
		return err
	}

	err = linter.New(
		linter.WithTrusted(
			c.Bool("trusted"),
		),
	).Lint(conf)

	if err != nil {
		return err
	}

	fmt.Println("Lint complete. Yaml file is valid")
	return nil
}
