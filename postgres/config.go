package postgres

import (
	"github.com/thanhminhmr/go-common/configuration"
)

// Config defines the options that are used when connecting to a PostgreSQL instance
type Config struct {
	Address      string `env:"POSTGRES_ADDRESS" validate:"required,hostname_port"`
	Username     string `env:"POSTGRES_USER" validate:"required"`
	Password     string `env:"POSTGRES_PASSWORD" validate:"required"`
	DatabaseName string `env:"POSTGRES_DB_NAME" validate:"required"`
}

// ExtraConfig defines extra options that are used when connecting to a PostgreSQL instance
type ExtraConfig struct {
	SSLMode           string `env:"POSTGRES_SSL_MODE"`
	SSLCert           string `env:"POSTGRES_SSL_CERT"`
	SSLKey            string `env:"POSTGRES_SSL_KEY"`
	SSLRootCert       string `env:"POSTGRES_SSL_ROOT_CERT"`
	ConnectionTimeout uint   `env:"POSTGRES_CONNECTION_TIMEOUT" validate:"min=0,max=60"`
	MinConnections    uint   `env:"POSTGRES_MIN_CONNECTIONS" validate:"min=0,max=256"`
	MaxConnections    uint   `env:"POSTGRES_MAX_CONNECTIONS" validate:"min=0,max=256"`
	MaxRetry          uint   `env:"POSTGRES_MAX_RETRY" validate:"min=0,max=1000"`
	RetryInterval     uint   `env:"POSTGRES_RETRY_INTERVAL" validate:"min=1,max=3600"`
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
