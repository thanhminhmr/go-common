package postgres

import (
	"github.com/thanhminhmr/go-common/configuration"
)

// Config defines the options that are used when connecting to a PostgreSQL instance
type Config struct {
	Host         string `env:"POSTGRES_HOST"`
	Port         string `env:"POSTGRES_PORT"`
	Username     string `env:"POSTGRES_USER"`
	Password     string `env:"POSTGRES_PASSWORD"`
	DatabaseName string `env:"POSTGRES_DB_NAME"`
}

// ExtraConfig defines extra options that are used when connecting to a PostgreSQL instance
type ExtraConfig struct {
	SSLMode           string `env:"POSTGRES_SSL_MODE"`
	SSLCert           string `env:"POSTGRES_SSL_CERT"`
	SSLKey            string `env:"POSTGRES_SSL_KEY"`
	SSLRootCert       string `env:"POSTGRES_SSL_ROOT_CERT"`
	ConnectionTimeout int32  `env:"POSTGRES_CONNECTION_TIMEOUT"`
	MinConnections    int32  `env:"POSTGRES_MIN_CONNECTIONS"`
	MaxConnections    int32  `env:"POSTGRES_MAX_CONNECTIONS"`
	MaxRetry          uint64 `env:"POSTGRES_MAX_RETRY"`
	RetryInterval     uint64 `env:"POSTGRES_RETRY_INTERVAL"`
	LogLevel          string `env:"POSTGRES_LOG_LEVEL"`
}

func init() {
	configuration.SetDefault("POSTGRES_CONNECTION_TIMEOUT", "15")
	configuration.SetDefault("POSTGRES_MIN_CONNECTIONS", "4")
	configuration.SetDefault("POSTGRES_MAX_CONNECTIONS", "128")
	configuration.SetDefault("POSTGRES_MAX_RETRY", "30")
	configuration.SetDefault("POSTGRES_RETRY_INTERVAL", "5")
	configuration.SetDefault("POSTGRES_LOG_LEVEL", "info")
}
