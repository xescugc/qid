package config

type Config struct {
	Port int `koanf:"port"`

	DBSystem string `koanf:"db-system"`

	// MySQL
	DBHost     string `koanf:"db-host"`
	DBPort     int    `koanf:"db-port"`
	DBUser     string `koanf:"db-user"`
	DBPassword string `koanf:"db-password"`
	DBName     string `koanf:"db-name"`

	RunWorker   bool `koanf:"run-worker"`
	Concurrency int  `koanf:"concurrency"`

	PubSubSystem string `koanf:"pubsub-system"`
}
