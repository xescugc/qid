package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/xescugc/qid/qid/utils"
)

const (
	userPasswordSeparator = ":"
)

var (
	userPasswordCmd = &cli.Command{
		Name:  "user-password",
		Usage: "This is a helper if you want to add users via configuration. I generates the right value to set",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Aliases: []string{"u"}, Required: true, Usage: "Username of the user"},
			&cli.StringFlag{Name: "password", Aliases: []string{"p"}, Required: true, Usage: "Plain password of the user"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			hash, err := utils.HashPassword(cmd.String("password"))
			if err != nil {
				return err
			}

			fmt.Printf("%s%s%s\n", cmd.String("username"), userPasswordSeparator, hash)
			return nil
		},
	}
)
