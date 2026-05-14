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
		Usage: "PikoCI is a small CI/CD build on top of a Queue(Pub/Sub) system",
		Commands: []*cli.Command{
			serverCmd,
			clientCmd,
			workerCmd,
			userPasswordCmd,
		},
	}
)
