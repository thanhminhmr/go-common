package http

import "github.com/thanhminhmr/go-common/configuration"

type ServerConfig struct {
	Port uint16 `env:"HTTP_SERVER_PORT"`
}

type ServerExtraConfig struct {
	ReadHeaderTimeout uint32 `env:"HTTP_SERVER_READ_HEADER_TIMEOUT"`
	IdleTimeout       uint32 `env:"HTTP_SERVER_IDLE_TIMEOUT"`
	MaxHeaderBytes    uint32 `env:"HTTP_SERVER_MAX_HEADER_BYTES"`
}

func init() {
	configuration.SetDefault("HTTP_SERVER_READ_HEADER_TIMEOUT", "5")
	configuration.SetDefault("HTTP_SERVER_IDLE_TIMEOUT", "60")
	configuration.SetDefault("HTTP_SERVER_MAX_HEADER_BYTES", "4096")
}
