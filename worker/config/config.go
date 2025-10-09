package config

type Config struct {
	QIDURL string `koanf:"qid-url"`

	Concurrency  int    `koanf:"concurrency"`
	PubSubSystem string `koanf:"pubsub-system"`
}
