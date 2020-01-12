package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"marwan.io/clientgen/clientgen"
)

func main() {
	app := &cli.App{
		Name:        "clientgen",
		Usage:       "clientgen [options]",
		Description: "generate Go clients from GraphQL Schema Files",
		Action: func(ctx *cli.Context) error {
			err := run(ctx)
			if err != nil {
				return cli.Exit(err, 1)
			}
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "out",
				Aliases: []string{"o"},
				Usage:   "destination for outputting the client go file",
				Value:   "clientgen.go",
			},
		},
	}
	app.Run(os.Args)
}

func run(ctx *cli.Context) error {
	cfg := &config.Config{
		Exec:           config.PackageConfig{Filename: "generated.go"},
		SchemaFilename: config.StringList{"*.graphql"},
		Model:          config.PackageConfig{Filename: "models_gen.go"},
	}
	err := processConfig(cfg)
	if err != nil {
		return fmt.Errorf("error processing graphql schemas: %v", err)
	}
	m := modelgen.New()
	err = m.(plugin.ConfigMutator).MutateConfig(cfg)
	if err != nil {
		return fmt.Errorf("error generating go models: %v", err)
	}
	data, err := codegen.BuildData(cfg)
	if err != nil {
		return fmt.Errorf("error compiling go type information: %v", err)
	}
	dest := ctx.String("out")
	err = clientgen.New(dest, "").(plugin.CodeGenerator).GenerateCode(data)
	if err != nil {
		return fmt.Errorf("error generating go client: %v", err)
	}
	return nil
}

func processConfig(cfg *config.Config) error {
	var path2regex = strings.NewReplacer(
		`.`, `\.`,
		`*`, `.+`,
		`\`, `[\\/]`,
		`/`, `[\\/]`,
	)
	preGlobbing := cfg.SchemaFilename
	cfg.SchemaFilename = config.StringList{}
	for _, f := range preGlobbing {
		var matches []string

		// for ** we want to override default globbing patterns and walk all
		// subdirectories to match schema files.
		if strings.Contains(f, "**") {
			pathParts := strings.SplitN(f, "**", 2)
			rest := strings.TrimPrefix(strings.TrimPrefix(pathParts[1], `\`), `/`)
			// turn the rest of the glob into a regex, anchored only at the end because ** allows
			// for any number of dirs in between and walk will let us match against the full path name
			globRe := regexp.MustCompile(path2regex.Replace(rest) + `$`)

			if err := filepath.Walk(pathParts[0], func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if globRe.MatchString(strings.TrimPrefix(path, pathParts[0])) {
					matches = append(matches, path)
				}

				return nil
			}); err != nil {
				return errors.Wrapf(err, "failed to walk schema at root %s", pathParts[0])
			}
		} else {
			var err error
			matches, err = filepath.Glob(f)
			if err != nil {
				return errors.Wrapf(err, "failed to glob schema filename %s", f)
			}
		}

		for _, m := range matches {
			if cfg.SchemaFilename.Has(m) {
				continue
			}
			cfg.SchemaFilename = append(cfg.SchemaFilename, m)
		}
	}
	return nil
}
