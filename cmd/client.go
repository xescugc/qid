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
	"github.com/spf13/cobra"
	"github.com/xescugc/pikoci/pikoci"
	"github.com/xescugc/pikoci/pikoci/transport/http/client"
)

var (
	configAuthenticationPath = "pikoci/authentication"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Interacts with the PikoCI server",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		jwt, _ := cmd.Flags().GetString("jwt")
		if jwt == "" {
			configFilePath, err := xdg.SearchConfigFile(configAuthenticationPath)
			if err != nil {
				return nil
			}

			data, err := os.ReadFile(configFilePath)
			if err != nil {
				return nil
			}
			cmd.Flags().Set("jwt", string(data))
		}
		return nil
	},
}

func init() {
	clientCmd.PersistentFlags().StringP("url", "u", "localhost:4000", "URL to the PikoCI server")
	clientCmd.MarkPersistentFlagRequired("url")
	clientCmd.PersistentFlags().String("jwt", "", "Provide the JWT to authenticate on the API, if not provided will read it from the FS")

	clientCmd.AddCommand(loginCmd)
	clientCmd.AddCommand(pipelinesCmd)
	clientCmd.AddCommand(jobsCmd)
}

// login
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: fmt.Sprintf("Logs the User in and stores the JWT locally at %q", filepath.Join(xdg.ConfigHome, configAuthenticationPath)),
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")

		c, err := client.New(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		_, jwtToken, err := c.UserLogin(cmd.Context(), username, password)
		if err != nil {
			return fmt.Errorf("failed to log in: %w", err)
		}

		configFilePath, err := xdg.ConfigFile(configAuthenticationPath)
		if err != nil {
			return fmt.Errorf("failed to check $XDG_CONFIG_HOME: %w", err)
		}

		err = os.WriteFile(configFilePath, []byte(jwtToken), 0666)
		if err != nil {
			return fmt.Errorf("failed to write the authentication file: %w", err)
		}

		fmt.Println("Login successfully")
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("username", "u", "", "Username use to login")
	loginCmd.Flags().StringP("password", "p", "", "Password use to login")
	loginCmd.MarkFlagRequired("username")
	loginCmd.MarkFlagRequired("password")
}

// pipelines
var pipelinesCmd = &cobra.Command{
	Use:   "pipelines",
	Short: "Interacts with the PikoCI Pipelines",
}

func init() {
	pipelinesCmd.PersistentFlags().String("team-canonical", "main", "Team Canonical to scope the action")
	pipelinesCmd.MarkPersistentFlagRequired("team-canonical")

	pipelinesCmd.AddCommand(pipelinesCreateCmd)
	pipelinesCmd.AddCommand(pipelinesUpdateCmd)
	pipelinesCmd.AddCommand(pipelinesListCmd)
	pipelinesCmd.AddCommand(pipelinesGetCmd)
	pipelinesCmd.AddCommand(pipelinesGraphCmd)
	pipelinesCmd.AddCommand(pipelinesDeleteCmd)
}

var pipelinesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates a new PikoCI Pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")
		name, _ := cmd.Flags().GetString("name")
		configPath, _ := cmd.Flags().GetString("config")
		varsPath, _ := cmd.Flags().GetString("vars")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		f, err := os.Open(configPath)
		if err != nil {
			return fmt.Errorf("failed to open config file at %q: %w", configPath, err)
		}
		defer f.Close()

		b, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("failed to read config file at %q: %w", configPath, err)
		}

		var vars map[string]interface{}
		if varsPath != "" {
			vf, err := os.Open(varsPath)
			if err != nil {
				return fmt.Errorf("failed to open vars file at %q: %w", varsPath, err)
			}
			defer vf.Close()

			err = json.NewDecoder(vf).Decode(&vars)
			if err != nil {
				return fmt.Errorf("failed to read decode vars file at %q: %w", varsPath, err)
			}
		}

		_, err = c.CreatePipeline(cmd.Context(), tc, name, b, vars)
		if err != nil {
			return fmt.Errorf("failed to create Pipeline %q: %w", name, err)
		}

		return nil
	},
}

func init() {
	pipelinesCreateCmd.Flags().StringP("name", "n", "", "Name of the Pipeline")
	pipelinesCreateCmd.Flags().StringP("config", "c", "", "Path to the Pipeline config file")
	pipelinesCreateCmd.Flags().StringP("vars", "v", "", "Path to the Pipeline var file (JSON)")
	pipelinesCreateCmd.MarkFlagRequired("name")
	pipelinesCreateCmd.MarkFlagRequired("config")
}

var pipelinesUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates a PikoCI Pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")
		name, _ := cmd.Flags().GetString("name")
		configPath, _ := cmd.Flags().GetString("config")
		varsPath, _ := cmd.Flags().GetString("vars")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}
		err = createPipeline(cmd.Context(), c, tc, name, configPath, varsPath)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("public") {
			public, _ := cmd.Flags().GetBool("public")
			err = c.SetPipelinePublic(cmd.Context(), tc, name, public)
			if err != nil {
				return fmt.Errorf("failed to set pipeline public: %w", err)
			}
		}
		return nil
	},
}

func init() {
	pipelinesUpdateCmd.Flags().StringP("name", "n", "", "Name of the Pipeline")
	pipelinesUpdateCmd.Flags().StringP("config", "c", "", "Path to the Pipeline config file")
	pipelinesUpdateCmd.Flags().StringP("vars", "v", "", "Path to the Pipeline var file (JSON)")
	pipelinesUpdateCmd.Flags().Bool("public", false, "Make the pipeline publicly visible")
	pipelinesUpdateCmd.MarkFlagRequired("name")
	pipelinesUpdateCmd.MarkFlagRequired("config")
}

var pipelinesListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists the PikoCI Pipelines",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		pps, err := c.ListPipelines(cmd.Context(), tc)
		if err != nil {
			return fmt.Errorf("failed to list Pipelines: %w", err)
		}

		spew.Dump(pps)
		return nil
	},
}

var pipelinesGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Gets a PikoCI Pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")
		name, _ := cmd.Flags().GetString("name")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		pp, err := c.GetPipeline(cmd.Context(), tc, name)
		if err != nil {
			return fmt.Errorf("failed to get Pipeline %q: %w", name, err)
		}

		spew.Dump(pp)
		return nil
	},
}

func init() {
	pipelinesGetCmd.Flags().StringP("name", "n", "", "Name of the Pipeline")
	pipelinesGetCmd.MarkFlagRequired("name")
}

var pipelinesGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Outputs the pipeline graph in DOT format",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")
		name, _ := cmd.Flags().GetString("name")
		format, _ := cmd.Flags().GetString("format")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		image, err := c.GetPipelineImage(cmd.Context(), tc, name, format)
		if err != nil {
			return fmt.Errorf("failed to get pipeline graph for %q: %w", name, err)
		}

		fmt.Fprint(os.Stdout, string(image))
		return nil
	},
}

func init() {
	pipelinesGraphCmd.Flags().StringP("name", "n", "", "Name of the Pipeline")
	pipelinesGraphCmd.Flags().StringP("format", "f", "dot", "Output format (dot)")
	pipelinesGraphCmd.MarkFlagRequired("name")
}

var pipelinesDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes a PikoCI Pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")
		name, _ := cmd.Flags().GetString("name")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		err = c.DeletePipeline(cmd.Context(), tc, name)
		if err != nil {
			return fmt.Errorf("failed to delete Pipeline %q: %w", name, err)
		}

		return nil
	},
}

func init() {
	pipelinesDeleteCmd.Flags().StringP("name", "n", "", "Name of the Pipeline")
	pipelinesDeleteCmd.MarkFlagRequired("name")
}

// jobs
var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Interacts with the PikoCI Jobs",
}

func init() {
	jobsCmd.PersistentFlags().String("team-canonical", "", "Team Canonical to scope the action")
	jobsCmd.PersistentFlags().String("pipeline-name", "", "Name of the Pipeline")
	jobsCmd.MarkPersistentFlagRequired("team-canonical")
	jobsCmd.MarkPersistentFlagRequired("pipeline-name")

	jobsCmd.AddCommand(jobsGetCmd)
	jobsCmd.AddCommand(jobsTriggerCmd)
}

var jobsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Gets a PikoCI Pipeline Job",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")
		pn, _ := cmd.Flags().GetString("pipeline-name")
		jn, _ := cmd.Flags().GetString("job-name")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		j, err := c.GetPipelineJob(cmd.Context(), tc, pn, jn)
		if err != nil {
			return fmt.Errorf("failed to get Job %q from Pipeline %q: %w", jn, pn, err)
		}

		spew.Dump(j)
		return nil
	},
}

func init() {
	jobsGetCmd.Flags().StringP("job-name", "n", "", "Name of the Job")
	jobsGetCmd.MarkFlagRequired("job-name")
}

var jobsTriggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Triggers a new PikoCI Pipeline Job",
	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		jwt, _ := cmd.Flags().GetString("jwt")
		tc, _ := cmd.Flags().GetString("team-canonical")
		pn, _ := cmd.Flags().GetString("pipeline-name")
		jn, _ := cmd.Flags().GetString("job-name")

		c, err := newClientWithConfig(url, jwt)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", url, err)
		}

		err = c.TriggerPipelineJob(cmd.Context(), tc, pn, jn)
		if err != nil {
			return fmt.Errorf("failed to trigger Job %q from Pipeline %q: %w", jn, pn, err)
		}

		return nil
	},
}

func init() {
	jobsTriggerCmd.Flags().StringP("job-name", "n", "", "Name of the Job")
	jobsTriggerCmd.MarkFlagRequired("job-name")
}

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
