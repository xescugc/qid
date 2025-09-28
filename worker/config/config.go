package config

type Config struct {
	RedisAddr string `koanf:"redis-addr"`
	QIDURL    string `koanf:"qid-url"`
}
