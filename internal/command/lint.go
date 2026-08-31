package command

import (
	"github.com/urfave/cli/v2"
)

// LintCommand is a command for linting yaml tpl files
type LintCommand struct{}

// Command returns a cli.Command instance
func (c LintCommand) Command() *cli.Command {
	return &cli.Command{
		Name:      "lint",
		Usage:     "validate yaml gotpl formatting (no changes written)",
		UsageText: "Example: gotpl-linter --vv 10 lint --path ./templates --filter '*.yaml'",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:        "path",
				Aliases:     []string{"p"},
				Usage:       "path to go tpl files",
				DefaultText: "./",
				Required:    false,
			},
			&cli.StringFlag{
				Name:     "filter",
				Aliases:  []string{"f"},
				Usage:    "filter files by pattern",
				Required: false,
				Value:    "*",
			},
			&cli.BoolFlag{
				Name:     "show",
				Aliases:  []string{"s"},
				Usage:    "output expected formatting",
				Required: false,
				Value:    false,
			},
			&cli.BoolFlag{
				Name:     "trim",
				Aliases:  []string{"t"},
				Usage:    "require non-{{- template openings to be {{- so they can be re-indented",
				Required: false,
				Value:    false,
			},
		},
		Action: c.lintAction,
	}
}

func (c LintCommand) lintAction(ctx *cli.Context) error {
	path := ctx.Path("path")
	filter := ctx.String("filter")
	output := ctx.Bool("show")
	forceTrim := ctx.Bool("trim")

	return yamlTplLinting(path, filter, false, output, forceTrim)
}
