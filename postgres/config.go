package postgres

import "github.com/thanhminhmr/go-common/configuration"

// Config defines the options that are used when connecting to a PostgreSQL instance
type Config struct {
	Address      string `env:"POSTGRES_ADDRESS" validate:"required,hostname_port"`
	Username     string `env:"POSTGRES_USER" validate:"required"`
	Password     string `env:"POSTGRES_PASSWORD" validate:"required"`
	DatabaseName string `env:"POSTGRES_DB_NAME" validate:"required"`
	LogLevel     string `env:"POSTGRES_LOG_LEVEL" validate:"oneof=trace debug info warn error none"`
}

func init() {
	configuration.SetDefault("POSTGRES_LOG_LEVEL", "info")
}
