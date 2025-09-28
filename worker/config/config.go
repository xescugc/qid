package config

type Config struct {
	QIDURL string `koanf:"qid-url"`

	PubSubSystem string `koanf:"pubsub-system"`
}
