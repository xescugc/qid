package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v3"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/transport/http/client"
	"github.com/xescugc/qid/worker"
	"github.com/xescugc/qid/worker/config"
	wpasynq "github.com/xescugc/qid/worker/providers/asynq"
)

var (
	workerCmd = &cli.Command{
		Name:  "worker",
		Usage: "Starts a QID Worker",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the config file"},

			&cli.StringFlag{Name: "qid-url", Aliases: []string{"u"}, Value: "localhost:4000", Usage: "URL to the QID server"},
			&cli.StringFlag{Name: "redis-addr", Value: "localhost:6379", Usage: "Redis Address"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			k := koanf.New(".")

			// Load the config files provided in the commandline.
			cfgf := cmd.String("config")
			if cfgf != "" {
				if err := k.Load(file.Provider(cfgf), json.Parser()); err != nil {
					return fmt.Errorf("error loading file: %v", err)
				}
			}
			if err := k.Load(cliflagv3.Provider(cmd, "qid.worker"), nil); err != nil {
				return fmt.Errorf("error loading flags: %v", err)
			}
			if err := k.Load(env.Provider(".", env.Opt{
				TransformFunc: func(k, v string) (string, any) {
					// Transform the key.
					k = strings.ReplaceAll(strings.ToLower(k), "_", "-")

					// Transform the value into slices, if they contain spaces.
					// Eg: MYVAR_TAGS="foo bar baz" -> tags: ["foo", "bar", "baz"]
					// This is to demonstrate that string values can be transformed to any type
					// where necessary.
					if strings.Contains(v, " ") {
						return k, strings.Split(v, " ")
					}

					return fmt.Sprintf("qid.worker.%s", k), v
				},
			}), nil); err != nil {
				return fmt.Errorf("error loading environments: %v", err)
			}

			var cfg config.Config
			k.Unmarshal("qid.worker", &cfg)

			c, err := client.New(cfg.QIDURL)
			if err != nil {
				return fmt.Errorf("failed to initialize client with url %q: %w", cfg.QIDURL, err)
			}
			//srv := asynq.NewServer(
			//asynq.RedisClientOpt{Addr: cfg.RedisAddr},
			//)

			//w := worker.New(c)

			//// Use asynq.HandlerFunc adapter for a handler function
			//if err := srv.Run(asynq.HandlerFunc(wpasynq.Handler(w))); err != nil {
			//log.Fatal(err)
			//}

			runWorker(c, cfg.RedisAddr)

			return nil
		},
	}
)

func runWorker(s qid.Service, ra string) {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: ra},
		asynq.Config{},
	)

	w := worker.New(s)

	// Use asynq.HandlerFunc adapter for a handler function
	if err := srv.Run(asynq.HandlerFunc(wpasynq.Handler(w))); err != nil {
		log.Fatal(err)
	}
}
