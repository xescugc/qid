package config

type Config struct {
	PikoCIURL string `koanf:"pikoci-url"`
	JWTSecret []byte `koanf:"jwt-secret"`

	Concurrency  int    `koanf:"concurrency"`
	PubSubSystem string `koanf:"pubsub-system"`

	LogLevel string `koanf:"log-level"`
}
