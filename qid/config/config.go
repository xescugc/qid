package config

type Config struct {
	Port      int    `koanf:"port"`
	RedisAddr string `koanf:"redis-addr"`

	DBHost     string `koanf:"db-host"`
	DBPort     int    `koanf:"db-port"`
	DBUser     string `koanf:"db-user"`
	DBPassword string `koanf:"db-password"`
	DBName     string `koanf:"db-name"`

	RunWorker bool `koanf:"run-worker"`
}
