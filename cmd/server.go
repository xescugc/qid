package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cycloidio/sqlr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	"github.com/urfave/cli/v3"
	"github.com/xescugc/pikoci/pikoci"
	"github.com/xescugc/pikoci/pikoci/config"
	"github.com/xescugc/pikoci/pikoci/mysql"
	"github.com/xescugc/pikoci/pikoci/mysql/migrate"
	tshttp "github.com/xescugc/pikoci/pikoci/transport/http"
	"github.com/xescugc/pikoci/pikoci/unitwork"
	"github.com/xescugc/pikoci/pikoci/user"
	"gocloud.dev/pubsub"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"gocloud.dev/pubsub/mempubsub"
	"gocloud.dev/pubsub/natspubsub"

	_ "gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/rabbitpubsub"

	"github.com/adrg/xdg"
)

var mainTeamCanonical = "main"

var (
	serverCmd = &cli.Command{
		Name:  "server",
		Usage: "Starts the PikoCI server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the config file"},

			&cli.IntFlag{Name: "port", Aliases: []string{"p"}, Value: 8080, Usage: "Port in which to start the server"},

			&cli.StringFlag{Name: "jwt-secret", Required: true, Usage: "Declares the Secret used to sign the JWT when user login"},

			&cli.StringSliceFlag{Name: "users", Usage: "List of Users which will have 'USERNAME:HASH-PASSWORD', you can use the 'user-password' command to help you"},

			&cli.StringFlag{Name: "db-system", Value: mysql.Mem, Usage: "Which DB system to use (mem, sqlite, mysql, postgresql)"},
			&cli.StringFlag{Name: "db-host", Usage: "Database Host"},
			&cli.IntFlag{Name: "db-port", Usage: "Database Port"},
			&cli.StringFlag{Name: "db-user", Usage: "Database User"},
			&cli.StringFlag{Name: "db-password", Usage: "Database Password"},
			&cli.StringFlag{Name: "db-name", Usage: "Database Name"},
			&cli.BoolFlag{Name: "run-migrations", Value: true, Usage: "Flag to know if migrations should be ran"},

			&cli.BoolFlag{Name: "run-worker", Value: true, Usage: "Runs a worker with PikoCI server"},
			&cli.IntFlag{Name: "concurrency", Value: 1, Usage: "Number of workers to start in one instance"},

			&cli.StringFlag{Name: "pubsub-system", Value: mempubsub.Scheme, Usage: "Which PubSub system to use (mem, nats, rabbit, kafka). Env vars: NATS_SERVER_URL, RABBIT_SERVER_URL, KAFKA_BROKERS"},

			&cli.StringFlag{Name: "log-level", Value: "info", Usage: "Sets the log level ('debug', 'info', 'warn', 'error')"},

			&cli.StringFlag{Name: "team-canonical", Aliases: []string{"tc"}, Value: mainTeamCanonical, Usage: "Team Canonical to scope the action", Local: true},
			&cli.StringFlag{Name: "pipeline-config", Aliases: []string{"c"}, Usage: "Path to the Pipeline config file", TakesFile: true},
			&cli.StringFlag{Name: "pipeline-vars", Aliases: []string{"v"}, Usage: "Path to the Pipeline var file (JSON)", TakesFile: true},
			&cli.StringFlag{Name: "pipeline-name", Aliases: []string{"n", "pn"}, Usage: "Name of the Pipeline"},
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

			if err := k.Load(cliflagv3.Provider(cmd, "pikoci.server"), nil); err != nil {
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

					return fmt.Sprintf("pikoci.server.%s", k), v
				},
			}), nil); err != nil {
				return fmt.Errorf("error loading environments: %v", err)
			}

			var cfg config.Config
			k.Unmarshal("pikoci.server", &cfg)

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseSlogLevel(cfg.LogLevel)}))
			logger = logger.With("service", "pikoci")

			if cfg.DBSystem != mysql.Mem && cfg.DBSystem != mysql.MySQL && cfg.DBSystem != mysql.SQLite && cfg.DBSystem != mysql.PostgreSQL {
				return fmt.Errorf("invalid DBSystem %q, should be one of: %s, %s, %s or %s", cfg.DBSystem, mysql.Mem, mysql.MySQL, mysql.SQLite, mysql.PostgreSQL)
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
			logger.Info("DB connection starting ...", "db-system", cfg.DBSystem)
			db, err := mysql.New(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, mysql.Options{
				DBName:          cfg.DBName,
				MultiStatements: true,
				ClientFoundRows: true,
				System:          cfg.DBSystem,
				DBFile:          dbFile,
			})
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			logger.Info("DB connection started", "db-system", cfg.DBSystem)

			if cmd.Bool("run-migrations") {
				logger.Info("Running migrations")
				err := migrate.Migrate(db, cfg.DBSystem)
				if err != nil {
					return fmt.Errorf("failed to run migrations: %w", err)
				}
				logger.Info("Migrations ran")
			}

			var querier sqlr.Querier = db
			if mysql.IsPostgreSQL(cfg.DBSystem) {
				querier = mysql.NewPGQuerier(db)
			}

			ur := mysql.NewUserRepository(querier)
			tr := mysql.NewTeamRepository(querier)
			ppr := mysql.NewPipelineRepository(querier)
			jr := mysql.NewJobRepository(querier)
			rr := mysql.NewResourceRepository(querier, cfg.DBSystem)
			rt := mysql.NewResourceTypeRepository(querier)
			br := mysql.NewBuildRepository(querier)
			rur := mysql.NewRunnerRepository(querier)
			str := mysql.NewSecretTypeRepository(querier)

			suow := unitwork.NewStartUnitOfWork(db, cfg.DBSystem)

			logger.Info("initializing service")
			var svc = pikoci.New(ctx, topic, ur, tr, ppr, jr, rr, rt, br, rur, str, suow, cfg.JWTSecret, logger)
			svc.StartScheduler(ctx)
			logger.Info("initialized service")

			logger.Info("initializing http handlers")
			var handler = tshttp.Handler(svc, cfg.JWTSecret, logger.With("component", "HTTP"))
			logger.Info("initialized http handlers")

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
				logger.Info("starting HTTP transport", "port", cfg.Port)
				errs <- svr.ListenAndServe()
			}()

			if cfg.RunWorker {
				logger.Info("Starting Worker ...")
				go func() {
					err := runWorker(ctx, cfg.PubSubSystem, topic, svc, cfg.Concurrency, cfg.LogLevel)
					errs <- fmt.Errorf("worker failed to start: %w", err)
				}()
			}

			if cmd.String("pipeline-name") != "" {
				err = createPipeline(ctx, svc, cmd.String("team-canonical"), cmd.String("pipeline-name"), cmd.String("pipeline-config"), cmd.String("pipeline-vars"))
				if err != nil {
					return err
				}
			}
			if users := cmd.StringSlice("users"); len(users) != 0 {
				for _, u := range users {
					us := strings.Split(u, userPasswordSeparator)
					isHashed := true
					_, err = svc.CreateUser(ctx, user.User{FullName: us[0], Username: us[0], Password: us[1]}, isHashed)
					if err != nil {
						return fmt.Errorf("failed to creat user %q: %w", us[0], err)
					}
				}
			}

			logger.Error("exit", "error", <-errs)

			return nil
		},
	}
)

func getSubscriptionURL(s string) string {
	u := fmt.Sprintf("%s://pikoci", s)
	switch s {
	case natspubsub.Scheme:
		u += "?queue=pikoci&natsv2"
	case "rabbit":
		// rabbit://pikoci — uses RABBIT_SERVER_URL env var
	case "kafka":
		u = "kafka://pikoci-group?topic=pikoci"
	}
	return u
}

func getTopicURL(s string) string {
	u := fmt.Sprintf("%s://pikoci", s)
	switch s {
	case natspubsub.Scheme:
		u += "?natsv2"
	case "rabbit":
		// rabbit://pikoci — uses RABBIT_SERVER_URL env var
	case "kafka":
		// kafka://pikoci — uses KAFKA_BROKERS env var
	}
	return u
}

func parseSlogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func generateWorkerJWT(js []byte) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"is_from_worker": true,
	})
	tokenString, err := token.SignedString(js)
	if err != nil {
		panic(err)
	}
	return tokenString
}
