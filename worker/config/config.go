package config

type Config struct {
	PikoCIURL string `mapstructure:"pikoci-url"`
	JWTSecret string `mapstructure:"jwt-secret"`

	Concurrency  int    `mapstructure:"concurrency"`
	PubSubSystem string `mapstructure:"pubsub-system"`

	LogLevel string `mapstructure:"log-level"`
}
