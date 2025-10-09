package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/cliflagv3"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/urfave/cli/v3"
	"github.com/xescugc/qid/qid"
	"github.com/xescugc/qid/qid/queue"
	"github.com/xescugc/qid/qid/transport/http/client"
	"github.com/xescugc/qid/worker"
	"github.com/xescugc/qid/worker/config"

	"github.com/go-kit/kit/log"

	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/mempubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

var (
	workerCmd = &cli.Command{
		Name:  "worker",
		Usage: "Starts a QID Worker",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Path to the config file"},

			&cli.StringFlag{Name: "qid-url", Aliases: []string{"u"}, Value: "localhost:4000", Usage: "URL to the QID server"},

			&cli.StringFlag{Name: "pubsub-system", Value: mempubsub.Scheme, Usage: "Which PubSub System to use"},

			&cli.IntFlag{Name: "concurrency", Value: 1, Usage: "Number of workers to start in one instance"},
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

			topic, err := pubsub.OpenTopic(ctx, getTopicURL(cfg.PubSubSystem))
			if err != nil {
				return fmt.Errorf("failed to open: %v", err)
			}
			defer topic.Shutdown(ctx)

			runWorker(ctx, cfg.PubSubSystem, topic, c, cfg.Concurrency)

			return nil
		},
	}
)

func runWorker(ctx context.Context, sy string, t queue.Topic, s qid.Service, c int) error {
	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)
	logger = log.With(logger, "service", "worker")
	// Create a subscription connected to that topic.
	subscription, err := pubsub.OpenSubscription(ctx, getSubscriptionURL(sy))
	if err != nil {
		return fmt.Errorf("failed to OpenSubscription: %w", err)
	}
	defer subscription.Shutdown(ctx)

	var wg sync.WaitGroup
	for i := range c {
		wg.Add(1)
		nlogger := log.With(logger, "num", i+1)
		nlogger.Log("msg", fmt.Sprintf("Starting Worker %d", i+1))
		w := worker.New(s, t, subscription, nlogger)

		go func() {
			err = w.Run(ctx)
			if err != nil {
				logger.Log("error", fmt.Errorf("failed to Run worker: %w", err).Error())
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return nil
}
