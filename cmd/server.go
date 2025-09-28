package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-kit/kit/log"
	"github.com/gorilla/handlers"
	"github.com/urfave/cli/v3"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/config"
	"github.com/xescugc/qid/qid/mysql"
	"github.com/xescugc/qid/qid/mysql/migrate"
	"github.com/xescugc/qid/qid/queue/providers/asynq"
	tshttp "github.com/xescugc/qid/qid/transport/http"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var (
	serverCmd = &cli.Command{
		Name:  "server",
		Usage: "Starts the QID server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the config file"},

			&cli.IntFlag{Name: "port", Aliases: []string{"p"}, Value: 8080, Usage: "Port in which to start the server"},

			&cli.StringFlag{Name: "db-host", Usage: "Database Host"},
			&cli.IntFlag{Name: "db-port", Usage: "Database Port"},
			&cli.StringFlag{Name: "db-user", Usage: "Database User"},
			&cli.StringFlag{Name: "db-password", Usage: "Database Password"},
			&cli.StringFlag{Name: "db-name", Usage: "Database Name"},
			&cli.BoolFlag{Name: "run-migrations", Value: true, Usage: "Flag to know if migrations should be ran"},

			&cli.StringFlag{Name: "redis-addr", Value: "localhost:6379", Usage: "Redis Address"},

			&cli.BoolFlag{Name: "run-worker", Value: true, Usage: "Runs a worker with QID server"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			k := koanf.New(".")
			var logger log.Logger
			logger = log.NewLogfmtLogger(os.Stderr)
			logger = log.With(logger, "ts", log.DefaultTimestampUTC)
			logger = log.With(logger, "caller", log.DefaultCaller)
			logger = log.With(logger, "service", "qid")

			// Load the config files provided in the commandline.
			cfgf := cmd.String("config")
			if cfgf != "" {
				if err := k.Load(file.Provider(cfgf), json.Parser()); err != nil {
					return fmt.Errorf("error loading file: %v", err)
				}
			}

			if err := k.Load(cliflagv3.Provider(cmd, "qid.server"), nil); err != nil {
				return fmt.Errorf("error loading flags: %v", err)
			}
			spew.Dump(k.All())
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

					return fmt.Sprintf("qid.server.%s", k), v
				},
			}), nil); err != nil {
				return fmt.Errorf("error loading environments: %v", err)
			}

			var cfg config.Config
			k.Unmarshal("qid.server", &cfg)

			q := asynq.New(cfg.RedisAddr)

			logger.Log("msg", "MariaDB connection starting ...")
			db, err := mysql.New(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, mysql.Options{
				DBName:          cfg.DBName,
				MultiStatements: true,
				ClientFoundRows: true,
			})
			if err != nil {
				panic(err)
			}
			logger.Log("msg", "MariaDB connection started")

			if cmd.Bool("run-migrations") {
				logger.Log("msg", "Running migrations")
				err := migrate.Migrate(db)
				if err != nil {
					panic(err)
				}
				logger.Log("msg", "Migrations ran")
			}

			ppr := mysql.NewPipelineRepository(db)
			jr := mysql.NewJobRepository(db)

			logger.Log("message", "initializing service")
			var svc = qid.New(q, ppr, jr)
			logger.Log("message", "initialized service")

			logger.Log("message", "initializing http handlers")
			var handler = tshttp.Handler(svc, log.With(logger, "component", "HTTP"))
			logger.Log("message", "initialized http handlers")

			mux := http.NewServeMux()
			mux.Handle("/", handler)
			//mux.Handle("/css/", http.FileServer(http.FS(assets.Assets)))
			//mux.Handle("/js/", http.FileServer(http.FS(assets.Assets)))

			svr := &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.Port),
				Handler: handlers.LoggingHandler(os.Stdout, mux),
			}

			errs := make(chan error)

			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
				errs <- fmt.Errorf("%s", <-c)
			}()

			go func() {
				logger.Log("transport", "HTTP", "port", cfg.Port)
				errs <- svr.ListenAndServe()
			}()

			if cfg.RunWorker {
				logger.Log("message", "Starting Worker ...")
				go func() {
					runWorker(svc, cfg.RedisAddr)
					errs <- fmt.Errorf("worker failed to start")
				}()
			}

			logger.Log("exit", <-errs)

			return nil
		},
	}
)
