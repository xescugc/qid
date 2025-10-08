package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/urfave/cli/v3"
	"github.com/xescugc/qid/qid/transport/http/client"
)

var (
	clientCmd = &cli.Command{
		Name:  "client",
		Usage: "Interacts with the QID server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "url", Aliases: []string{"u"}, Value: "localhost:4000", Usage: "URL to the QID server", Required: true, Local: true},
		},
		Commands: []*cli.Command{
			{
				Name:  "pipelines",
				Usage: "Interacts with the QID Pipelines",
				Commands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Creates a new QID Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
							&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the Pipeline config file", TakesFile: true, Required: true},
							&cli.StringFlag{Name: "vars", Aliases: []string{"v"}, Usage: "Path to the Pipeline var file (JSON)", TakesFile: true, Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := client.New(cmd.String("url"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							f, err := os.Open(cmd.String("config"))
							if err != nil {
								return fmt.Errorf("failed to open config file at %q: %w", cmd.String("config"), err)
							}
							defer f.Close()

							vf, err := os.Open(cmd.String("vars"))
							if err != nil {
								return fmt.Errorf("failed to open vars file at %q: %w", cmd.String("vars"), err)
							}
							defer vf.Close()

							b, err := io.ReadAll(f)
							if err != nil {
								return fmt.Errorf("failed to read config file at %q: %w", cmd.String("config"), err)
							}

							var vars map[string]interface{}
							err = json.NewDecoder(vf).Decode(&vars)
							if err != nil {
								return fmt.Errorf("failed to read decode vars file at %q: %w", cmd.String("vars"), err)
							}

							err = c.CreatePipeline(ctx, cmd.String("name"), b, vars)
							if err != nil {
								return fmt.Errorf("failed to create Pipeline %q: %w", cmd.String("name"), err)
							}

							return nil
						},
					},
					{
						Name:  "update",
						Usage: "Updates a QID Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
							&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the Pipeline config file", TakesFile: true, Required: true},
							&cli.StringFlag{Name: "vars", Aliases: []string{"v"}, Usage: "Path to the Pipeline var file (JSON)", TakesFile: true, Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := client.New(cmd.String("url"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							f, err := os.Open(cmd.String("config"))
							if err != nil {
								return fmt.Errorf("failed to open config file at %q: %w", cmd.String("config"), err)
							}
							defer f.Close()

							vf, err := os.Open(cmd.String("vars"))
							if err != nil {
								return fmt.Errorf("failed to open vars file at %q: %w", cmd.String("vars"), err)
							}
							defer vf.Close()

							b, err := io.ReadAll(f)
							if err != nil {
								return fmt.Errorf("failed to read config file at %q: %w", cmd.String("config"), err)
							}

							var vars map[string]interface{}
							err = json.NewDecoder(vf).Decode(&vars)
							if err != nil {
								return fmt.Errorf("failed to read decode vars file at %q: %w", cmd.String("vars"), err)
							}

							err = c.UpdatePipeline(ctx, cmd.String("name"), b, vars)
							if err != nil {
								return fmt.Errorf("failed to update Pipeline %q: %w", cmd.String("name"), err)
							}

							return nil
						},
					},
					{
						Name:  "list",
						Usage: "Lists the QID Pipelines",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := client.New(cmd.String("url"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							pps, err := c.ListPipelines(ctx)
							if err != nil {
								return fmt.Errorf("failed to list Pipelines: %w", err)
							}

							spew.Dump(pps)

							return nil
						},
					},
					{
						Name:  "get",
						Usage: "Get's a QID Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := client.New(cmd.String("url"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							pp, err := c.GetPipeline(ctx, cmd.String("name"))
							if err != nil {
								return fmt.Errorf("failed to create Pipeline %q: %w", cmd.String("name"), err)
							}

							spew.Dump(pp)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "Deletes a QID Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := client.New(cmd.String("url"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							err = c.DeletePipeline(ctx, cmd.String("name"))
							if err != nil {
								return fmt.Errorf("failed to delete Pipeline %q: %w", cmd.String("name"), err)
							}

							return nil
						},
					},
				},
			},
			{
				Name:  "jobs",
				Usage: "Interacts with the QID Jobs",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "pipeline-name", Aliases: []string{"pn"}, Usage: "Name of the Pipeline", Required: true, Local: true},
				},
				Commands: []*cli.Command{
					{
						Name:  "get",
						Usage: "Get's a QID Pipeline Job",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "job-name", Aliases: []string{"n", "jn"}, Usage: "Name of the Job", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := client.New(cmd.String("url"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							j, err := c.GetPipelineJob(ctx, cmd.String("pipeline-name"), cmd.String("job-name"))
							if err != nil {
								return fmt.Errorf("failed to get Job %q from Pipeline %q: %w", cmd.String("job-name"), cmd.String("pipeline-name"), err)
							}

							spew.Dump(j)

							return nil
						},
					},
					{
						Name:  "trigger",
						Usage: "Triggers a new QID Pipeline Job",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "job-name", Aliases: []string{"n", "jn"}, Usage: "Name of the Job", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := client.New(cmd.String("url"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							err = c.TriggerPipelineJob(ctx, cmd.String("pipeline-name"), cmd.String("job-name"))
							if err != nil {
								return fmt.Errorf("failed to trigger Job %q from Pipeline %q: %w", cmd.String("job-name"), cmd.String("pipeline-name"), err)
							}

							return nil
						},
					},
				},
			},
		},
	}
)
