package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/davecgh/go-spew/spew"
	"github.com/urfave/cli/v3"
	"github.com/xescugc/pikoci/pikoci"
	"github.com/xescugc/pikoci/pikoci/transport/http/client"
)

var (
	configAuthenticationPath = "pikoci/authentication"
)

var (
	clientCmd = &cli.Command{
		Name:  "client",
		Usage: "Interacts with the PikoCI server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "url", Aliases: []string{"u"}, Value: "localhost:4000", Usage: "URL to the PikoCI server", Required: true, Local: true},
			&cli.StringFlag{Name: "jwt", Usage: "Provide the JWT to authenticate on the API, if not provided will read it from the FS", Local: true},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// If there is no on flags we try to set it from the configAuthenticationPath
			// if there is one. Which is set from a 'login'
			if cmd.String("jwt") == "" {
				configFilePath, err := xdg.SearchConfigFile(configAuthenticationPath)
				if err != nil {
					return ctx, nil
				}

				data, err := os.ReadFile(configFilePath)
				if err != nil {
					return ctx, nil
				}
				cmd.Set("jwt", string(data))
			}
			return nil, nil
		},
		Commands: []*cli.Command{
			{
				Name:  "login",
				Usage: fmt.Sprintf("Logs the User in and stores the JWT locally at %q", filepath.Join(xdg.ConfigHome, configAuthenticationPath)),
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "username", Aliases: []string{"u"}, Usage: "Username use to login", Required: true},
					&cli.StringFlag{Name: "password", Aliases: []string{"p"}, Usage: "Password use to login", Required: true},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					c, err := client.New(cmd.String("url"), cmd.String("jwt"))
					if err != nil {
						return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
					}

					_, jwt, err := c.UserLogin(ctx, cmd.String("username"), cmd.String("password"))
					if err != nil {
						return fmt.Errorf("failed to log in: %w", err)
					}

					configFilePath, err := xdg.ConfigFile(configAuthenticationPath)
					if err != nil {
						return fmt.Errorf("failed to check $XDG_CONFIG_HOME: %w", err)
					}

					err = os.WriteFile(configFilePath, []byte(jwt), 0666)
					if err != nil {
						return fmt.Errorf("failed to write the authentication file: %w", err)
					}

					println("Login successfully")

					return nil
				},
			},
			{
				Name:  "pipelines",
				Usage: "Interacts with the PikoCI Pipelines",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "team-canonical", Aliases: []string{"tc"}, Value: "main", Usage: "Team Canonical to scope the action", Required: true, Local: true},
				},
				Commands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Creates a new PikoCI Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
							&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the Pipeline config file", TakesFile: true, Required: true},
							&cli.StringFlag{Name: "vars", Aliases: []string{"v"}, Usage: "Path to the Pipeline var file (JSON)", TakesFile: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := newClientWithConfig(cmd.String("url"), cmd.String("jwt"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							f, err := os.Open(cmd.String("config"))
							if err != nil {
								return fmt.Errorf("failed to open config file at %q: %w", cmd.String("config"), err)
							}
							defer f.Close()

							b, err := io.ReadAll(f)
							if err != nil {
								return fmt.Errorf("failed to read config file at %q: %w", cmd.String("config"), err)
							}

							var vars map[string]interface{}
							if cmd.String("vars") != "" {
								vf, err := os.Open(cmd.String("vars"))
								if err != nil {
									return fmt.Errorf("failed to open vars file at %q: %w", cmd.String("vars"), err)
								}
								defer vf.Close()

								err = json.NewDecoder(vf).Decode(&vars)
								if err != nil {
									return fmt.Errorf("failed to read decode vars file at %q: %w", cmd.String("vars"), err)
								}
							}

							_, err = c.CreatePipeline(ctx, cmd.String("team-canonical"), cmd.String("name"), b, vars)
							if err != nil {
								return fmt.Errorf("failed to create Pipeline %q: %w", cmd.String("name"), err)
							}

							return nil
						},
					},
					{
						Name:  "update",
						Usage: "Updates a PikoCI Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
							&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the Pipeline config file", TakesFile: true, Required: true},
							&cli.StringFlag{Name: "vars", Aliases: []string{"v"}, Usage: "Path to the Pipeline var file (JSON)", TakesFile: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := newClientWithConfig(cmd.String("url"), cmd.String("jwt"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}
							return createPipeline(ctx, c, cmd.String("team-canonical"), cmd.String("name"), cmd.String("config"), cmd.String("vars"))
						},
					},
					{
						Name:  "list",
						Usage: "Lists the PikoCI Pipelines",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := newClientWithConfig(cmd.String("url"), cmd.String("jwt"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							pps, err := c.ListPipelines(ctx, cmd.String("team-canonical"))
							if err != nil {
								return fmt.Errorf("failed to list Pipelines: %w", err)
							}

							spew.Dump(pps)

							return nil
						},
					},
					{
						Name:  "get",
						Usage: "Get's a PikoCI Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := newClientWithConfig(cmd.String("url"), cmd.String("jwt"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							pp, err := c.GetPipeline(ctx, cmd.String("team-canonical"), cmd.String("name"))
							if err != nil {
								return fmt.Errorf("failed to get Pipeline %q: %w", cmd.String("name"), err)
							}

							spew.Dump(pp)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "Deletes a PikoCI Pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := newClientWithConfig(cmd.String("url"), cmd.String("jwt"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							err = c.DeletePipeline(ctx, cmd.String("team-canonical"), cmd.String("name"))
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
				Usage: "Interacts with the PikoCI Jobs",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "team-canonical", Aliases: []string{"tc"}, Usage: "Team Canonical to scope the action", Required: true, Local: true},
					&cli.StringFlag{Name: "pipeline-name", Aliases: []string{"pn"}, Usage: "Name of the Pipeline", Required: true, Local: true},
				},
				Commands: []*cli.Command{
					{
						Name:  "get",
						Usage: "Get's a PikoCI Pipeline Job",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "job-name", Aliases: []string{"n", "jn"}, Usage: "Name of the Job", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := newClientWithConfig(cmd.String("url"), cmd.String("jwt"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							j, err := c.GetPipelineJob(ctx, cmd.String("team-canonical"), cmd.String("pipeline-name"), cmd.String("job-name"))
							if err != nil {
								return fmt.Errorf("failed to get Job %q from Pipeline %q: %w", cmd.String("job-name"), cmd.String("pipeline-name"), err)
							}

							spew.Dump(j)

							return nil
						},
					},
					{
						Name:  "trigger",
						Usage: "Triggers a new PikoCI Pipeline Job",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "job-name", Aliases: []string{"n", "jn"}, Usage: "Name of the Job", Required: true},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							c, err := newClientWithConfig(cmd.String("url"), cmd.String("jwt"))
							if err != nil {
								return fmt.Errorf("failed to initialize client with url %q: %w", cmd.String("url"), err)
							}

							err = c.TriggerPipelineJob(ctx, cmd.String("team-canonical"), cmd.String("pipeline-name"), cmd.String("job-name"))
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

func newClientWithConfig(url, jwt string) (*client.Client, error) {
	c, err := client.New(url, jwt)
	if err != nil {
		return nil, err
	}
	configFilePath, err := xdg.ConfigFile(configAuthenticationPath)
	if err == nil {
		c.SetConfigPath(configFilePath)
	}
	return c, nil
}

func createPipeline(ctx context.Context, svc pikoci.Service, tc, name, config, vars string) error {

	f, err := os.Open(config)
	if err != nil {
		return fmt.Errorf("failed to open config file at %q: %w", config, err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read config file at %q: %w", config, err)
	}

	var vrs map[string]interface{}
	if vars != "" {
		vf, err := os.Open(vars)
		if err != nil {
			return fmt.Errorf("failed to open vars file at %q: %w", vars, err)
		}
		defer vf.Close()

		err = json.NewDecoder(vf).Decode(&vrs)
		if err != nil {
			return fmt.Errorf("failed to read decode vars file at %q: %w", vars, err)
		}
	}

	_, err = svc.CreatePipeline(ctx, tc, name, b, vrs)
	if err != nil {
		return fmt.Errorf("failed to create Pipeline %q: %w", name, err)
	}

	return nil
}
