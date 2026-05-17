package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xescugc/pikoci/pikoci"
	"github.com/xescugc/pikoci/pikoci/queue"
	"github.com/xescugc/pikoci/pikoci/transport/http/client"
	"github.com/xescugc/pikoci/worker"
	"github.com/xescugc/pikoci/worker/config"

	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/mempubsub"

	_ "gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/rabbitpubsub"
)

var workerViper = viper.New()

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Starts a PikoCI Worker",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Load config file if provided
		cfgFile, _ := cmd.Flags().GetString("config")
		if cfgFile != "" {
			workerViper.SetConfigFile(cfgFile)
			if err := workerViper.ReadInConfig(); err != nil {
				return fmt.Errorf("error loading config file: %v", err)
			}
		}

		var cfg config.Config
		if err := workerViper.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("error unmarshalling config: %v", err)
		}

		if cfg.JWTSecret == "" {
			return fmt.Errorf("required flag \"jwt-secret\" not set")
		}

		workerToken := generateWorkerJWT([]byte(cfg.JWTSecret))
		c, err := client.New(cfg.PikoCIURL, workerToken)
		if err != nil {
			return fmt.Errorf("failed to initialize client with url %q: %w", cfg.PikoCIURL, err)
		}

		topic, err := pubsub.OpenTopic(ctx, getTopicURL(cfg.PubSubSystem))
		if err != nil {
			return fmt.Errorf("failed to open: %v", err)
		}
		defer topic.Shutdown(ctx)

		runWorker(ctx, cfg.PubSubSystem, topic, c, cfg.Concurrency, cfg.LogLevel)

		return nil
	},
}

func init() {
	workerCmd.Flags().StringP("config", "c", "", "Path to the config file")
	workerCmd.Flags().StringP("pikoci-url", "u", "localhost:8080", "URL to the PikoCI server")
	workerCmd.Flags().String("pubsub-system", mempubsub.Scheme, "Which PubSub system to use (mem, nats, rabbit, kafka). Env vars: NATS_SERVER_URL, RABBIT_SERVER_URL, KAFKA_BROKERS")
	workerCmd.Flags().Int("concurrency", 1, "Number of workers to start in one instance")
	workerCmd.Flags().String("log-level", "info", "Sets the log level ('debug', 'info', 'warn', 'error')")
	workerCmd.Flags().String("jwt-secret", "", "JWT secret (must match the server's --jwt-secret)")

	workerViper.BindPFlags(workerCmd.Flags())

	workerViper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	workerViper.AutomaticEnv()
}

func runWorker(ctx context.Context, sy string, t queue.Topic, s pikoci.Service, c int, llvl string) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseSlogLevel(llvl)}))
	logger = logger.With("service", "worker")
	// Create a subscription connected to that topic.
	subscription, err := pubsub.OpenSubscription(ctx, getSubscriptionURL(sy))
	if err != nil {
		return fmt.Errorf("failed to OpenSubscription: %w", err)
	}
	defer subscription.Shutdown(ctx)

	var wg sync.WaitGroup
	for i := range c {
		wg.Add(1)
		nlogger := logger.With("num", i+1)
		nlogger.Info(fmt.Sprintf("Starting Worker %d", i+1))
		w := worker.New(s, t, subscription, nlogger)

		go func() {
			err = w.Run(ctx)
			if err != nil {
				logger.Error("failed to Run worker", "error", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return nil
}
