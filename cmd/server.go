package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/handlers"
	"github.com/urfave/cli/v3"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/config"
	"github.com/xescugc/qid/qid/mysql"
	"github.com/xescugc/qid/qid/mysql/migrate"
	tshttp "github.com/xescugc/qid/qid/transport/http"
	"gocloud.dev/pubsub"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"gocloud.dev/pubsub/mempubsub"
	"gocloud.dev/pubsub/natspubsub"

	"github.com/adrg/xdg"
)

var (
	serverCmd = &cli.Command{
		Name:  "server",
		Usage: "Starts the QID server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the config file"},

			&cli.IntFlag{Name: "port", Aliases: []string{"p"}, Value: 8080, Usage: "Port in which to start the server"},

			&cli.StringFlag{Name: "db-system", Value: mysql.Mem, Usage: "Flag to know if the database should run (mem, sqlite, mysql)"},
			&cli.StringFlag{Name: "db-host", Usage: "Database Host"},
			&cli.IntFlag{Name: "db-port", Usage: "Database Port"},
			&cli.StringFlag{Name: "db-user", Usage: "Database User"},
			&cli.StringFlag{Name: "db-password", Usage: "Database Password"},
			&cli.StringFlag{Name: "db-name", Usage: "Database Name"},
			&cli.BoolFlag{Name: "run-migrations", Value: true, Usage: "Flag to know if migrations should be ran"},

			&cli.BoolFlag{Name: "run-worker", Value: true, Usage: "Runs a worker with QID server"},
			&cli.IntFlag{Name: "concurrency", Value: 1, Usage: "Number of workers to start in one instance"},

			&cli.StringFlag{Name: "pubsub-system", Value: mempubsub.Scheme, Usage: "Which PubSub System to use"},

			&cli.StringFlag{Name: "log-level", Value: "info", Usage: "Sets the log level ('debug', 'info', 'warn', 'error')"},
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

			logger = level.NewFilter(logger, level.Allow(level.ParseDefault(cfg.LogLevel, level.InfoValue())))

			if cfg.DBSystem != mysql.Mem && cfg.DBSystem != mysql.MySQL && cfg.DBSystem != mysql.SQLite {
				return fmt.Errorf("invalid DBSystem %q, should be one of: %s, %s or %s", cfg.DBSystem, mysql.Mem, mysql.MySQL, mysql.SQLite)
			}

			topic, err := pubsub.OpenTopic(ctx, getTopicURL(cfg.PubSubSystem))
			if err != nil {
				return fmt.Errorf("failed to open: %v", err)
			}
			defer topic.Shutdown(ctx)
			dbFile, err := xdg.DataFile(filepath.Join(AppName, AppName+".db"))
			if err != nil {
				return fmt.Errorf("failed to create dbFile: %v", err)
			}
			level.Info(logger).Log("msg", "DB connection starting ...", "db-system", cfg.DBSystem)
			db, err := mysql.New(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, mysql.Options{
				DBName:          cfg.DBName,
				MultiStatements: true,
				ClientFoundRows: true,
				System:          cfg.DBSystem,
				DBFile:          dbFile,
			})
			if err != nil {
				panic(err)
			}
			level.Info(logger).Log("msg", "DB connection started", "db-system", cfg.DBSystem)

			if cmd.Bool("run-migrations") {
				level.Info(logger).Log("msg", "Running migrations")
				isSQLite := (cfg.DBSystem == mysql.Mem) || (cfg.DBSystem == mysql.SQLite)
				err := migrate.Migrate(db, isSQLite)
				if err != nil {
					panic(err)
				}
				level.Info(logger).Log("msg", "Migrations ran")
			}

			ppr := mysql.NewPipelineRepository(db)
			jr := mysql.NewJobRepository(db)
			rr := mysql.NewResourceRepository(db)
			rt := mysql.NewResourceTypeRepository(db)
			br := mysql.NewBuildRepository(db)
			rur := mysql.NewRunnerRepository(db)

			level.Info(logger).Log("message", "initializing service")
			var svc = qid.New(ctx, topic, ppr, jr, rr, rt, br, rur, logger)
			level.Info(logger).Log("message", "initialized service")

			level.Info(logger).Log("message", "initializing http handlers")
			var handler = tshttp.Handler(svc, log.With(logger, "component", "HTTP"))
			level.Info(logger).Log("message", "initialized http handlers")

			mux := http.NewServeMux()
			mux.Handle("/", handler)

			svr := &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.Port),
				Handler: handlers.CombinedLoggingHandler(os.Stdout, mux),
			}

			errs := make(chan error)

			go func() {
				c := make(chan os.Signal, 1)
				signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
				errs <- fmt.Errorf("%s", <-c)
			}()

			go func() {
				level.Info(logger).Log("transport", "HTTP", "port", cfg.Port)
				errs <- svr.ListenAndServe()
			}()

			if cfg.RunWorker {
				level.Info(logger).Log("message", "Starting Worker ...")
				go func() {
					err := runWorker(ctx, cfg.PubSubSystem, topic, svc, cfg.Concurrency, cfg.LogLevel)
					errs <- fmt.Errorf("worker failed to start: %w", err)
				}()
			}

			level.Error(logger).Log("exit", <-errs)

			return nil
		},
	}
)

func getSubscriptionURL(s string) string {
	u := fmt.Sprintf("%s://qid", s)
	switch s {
	case natspubsub.Scheme:
		u += "?queue=qid&natsv2"
	}
	return u
}

func getTopicURL(s string) string {
	u := fmt.Sprintf("%s://qid", s)
	switch s {
	case natspubsub.Scheme:
		u += "?natsv2"
	}
	return u
}
