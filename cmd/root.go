package cmd

import (
	"github.com/urfave/cli/v3"
)

var (
	Cmd = &cli.Command{
		Name:  "qid",
		Usage: "QID is a small CI/CD build on top of a Queue(Pub/Sub) system",
		Commands: []*cli.Command{
			serverCmd,
			clientCmd,
			workerCmd,
		},
	}
)
