package cmd

import (
	"github.com/urfave/cli/v3"
)

var (
	AppName = "pikoci"
)

var (
	Cmd = &cli.Command{
		Name:  AppName,
		Usage: "PikoCI is a self-hosted, portable CI/CD system",
		Commands: []*cli.Command{
			serverCmd,
			clientCmd,
			workerCmd,
			userPasswordCmd,
		},
	}
)
