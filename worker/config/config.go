package config

type Config struct {
	PikoCIURL   string `mapstructure:"pikoci-url"`
	WorkerToken string `mapstructure:"worker-token"`

	Concurrency  int    `mapstructure:"concurrency"`
	DrainTimeout string `mapstructure:"drain-timeout"`
	PubSubSystem string `mapstructure:"pubsub-system"`

	LogLevel string `mapstructure:"log-level"`
}
