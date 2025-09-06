package log

import (
	"os"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
}

func Init() *zerolog.Logger {
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02T15:04:05.000000000Z07:00",
	}).With().Timestamp().Caller().Logger()
	return &logger
}
