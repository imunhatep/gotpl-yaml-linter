package command

import (
	"errors"
	"github.com/imunhatep/gotpl-yaml-linter/internal/app"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// FormatCommand is a command for formatting yaml tpl files
type FormatCommand struct{}

// Command returns cli.Command for format command
func (c FormatCommand) Command() *cli.Command {
	return &cli.Command{
		Name:      "fmt",
		Usage:     "format yaml tpl files in place",
		UsageText: "Example: gotpl-linter --vv 10 fmt --path ./templates --filter '*.yaml'",
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
				Usage:    "rewrite non-{{- template openings to {{- so they can be re-indented (may change rendered output)",
				Required: false,
				Value:    false,
			},
		},
		Action: c.fmtAction,
	}
}

func (c FormatCommand) fmtAction(ctx *cli.Context) error {
	path := ctx.Path("path")
	filter := ctx.String("filter")
	output := ctx.Bool("show")
	forceTrim := ctx.Bool("trim")

	return yamlTplLinting(path, filter, true, output, forceTrim)
}

func yamlTplLinting(path, filter string, format, output, forceTrim bool) error {
	fileList, err := app.ListFilesInDir(path, filter)
	if err != nil {
		return err
	}

	log.Debug().Strs("files", fileList).Msg("files to lint")

	// Print matched files
	allTrue := true
	for _, match := range fileList {
		valid, err := app.FormatYamlTplFile(match, format, output, forceTrim)
		if err != nil {
			log.Error().Err(err).Str("file", match).Msg("failed formatting")
		}

		allTrue = allTrue && valid
	}

	if !allTrue {
		return errors.New("file formatting wasn't successful")
	}

	return nil
}
