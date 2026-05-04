package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	// Server
	Port string `envconfig:"PORT" default:"8080"`
	Env  string `envconfig:"ENV" default:"development"`

	// Database
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// JWT
	JWTSecret            string `envconfig:"JWT_SECRET" required:"true"`
	JWTAccessExpiryMin   int    `envconfig:"JWT_ACCESS_EXPIRY_MIN" default:"15"`
	JWTRefreshExpiryDays int    `envconfig:"JWT_REFRESH_EXPIRY_DAYS" default:"30"`

	// Storage — S3-compatible (R2 in prod, MinIO locally)
	StorageEndpoint  string `envconfig:"STORAGE_ENDPOINT" required:"true"`
	StorageBucket    string `envconfig:"STORAGE_BUCKET" required:"true"`
	StorageAccessKey string `envconfig:"STORAGE_ACCESS_KEY" required:"true"`
	StorageSecretKey string `envconfig:"STORAGE_SECRET_KEY" required:"true"`
	StorageRegion    string `envconfig:"STORAGE_REGION" default:"auto"`

	// Logging
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

func Load() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}
	return &c, nil
}
