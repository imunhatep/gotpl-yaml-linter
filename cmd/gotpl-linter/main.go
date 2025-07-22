package main

import (
	"context"
	"github.com/imunhatep/gotpl-yaml-linter/internal"
	command "github.com/imunhatep/gotpl-yaml-linter/internal/commands"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
)

var (
	BuildVersion = "development"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := internal.NewApp(BuildVersion)
	app.Commands = []*cli.Command{
		command.FormatCommand{}.Command(),
		command.LintCommand{}.Command(),
	}

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Error().Err(err).Msg("yaml tpl linting failed")
		os.Exit(1)
	}
}
