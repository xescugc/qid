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

	// SQLite
	DBFile string `koanf:"db-file"`

	RunWorker bool `koanf:"run-worker"`

	PubSubSystem string `koanf:"pubsub-system"`
}
