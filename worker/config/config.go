package config

type Config struct {
	QIDURL    string `koanf:"qid-url"`
	JWTSecret []byte `koanf:"jwt-secret"`

	Concurrency  int    `koanf:"concurrency"`
	PubSubSystem string `koanf:"pubsub-system"`

	LogLevel string `koanf:"log-level"`
}
