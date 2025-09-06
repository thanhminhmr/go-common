package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/thanhminhmr/go-common/errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
	"github.com/rubenv/sql-migrate"
	"github.com/sethvargo/go-retry"
)

// ConnectToPostgreSQL Connect to the PostgreSQL database that are specified in
// the environment variables. Migrate the database if required.
func ConnectToPostgreSQL(
	logger *zerolog.Logger,
	config *Config,
	extraConfig *ExtraConfig,
	migrationSource migrate.MigrationSource,
) (Database, error) {
	// parse configuration
	parsedConfig, err := pgxpool.ParseConfig(connectionURL(config, extraConfig))
	if err != nil {
		return nil, errors.String("Failed parsing PostgreSQL config").AddCause(err)
	}

	// set log level
	logLevel, err := tracelog.LogLevelFromString(extraConfig.LogLevel)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed parsing log level for PostgreSQL, default to level Info.")
		logLevel = tracelog.LogLevelInfo
	}

	// set other parameters
	parsedConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   tracelog.LoggerFunc(contextLogger),
		LogLevel: logLevel,
	}
	parsedConfig.MinConns = extraConfig.MinConnections
	parsedConfig.MaxConns = extraConfig.MaxConnections

	// try connect
	var pool *pgxpool.Pool
	backoff := retry.WithMaxRetries(
		extraConfig.MaxRetry,
		retry.NewConstant(time.Duration(extraConfig.RetryInterval)*time.Second),
	)
	if err := retry.Do(context.Background(), backoff, retryConnect(logger, parsedConfig, &pool)); err != nil {
		logger.Error().Err(err).Msg("Failed connecting to PostgreSQL!")
		return nil, errors.String("Failed connecting to PostgreSQL!").AddCause(err)
	}

	// migrate database
	if migrationSource != nil {
		_, err := migrate.Exec(stdlib.OpenDBFromPool(pool), "postgres", migrationSource, migrate.Up)
		if err != nil {
			logger.Error().Err(err).Msg("Failed migrating PostgreSQL database!")
			return nil, errors.String("Failed migrating PostgreSQL database!").AddCause(err)
		}
	}

	// return connection
	return &_database[*pgxpool.Pool]{
		_connection: _connection[*pgxpool.Pool]{
			pgx: pool,
		},
	}, nil
}

func connectionURL(config *Config, extraConfig *ExtraConfig) string {
	host := config.Host
	if port := config.Port; port != "" {
		host = host + ":" + port
	}
	targetUrl := &url.URL{
		Scheme: "postgres",
		Host:   host,
		Path:   config.DatabaseName,
	}
	if config.Username != "" || config.Password != "" {
		targetUrl.User = url.UserPassword(config.Username, config.Password)
	}
	query := targetUrl.Query()
	if timeout := extraConfig.ConnectionTimeout; timeout > 0 {
		query.Add("connect_timeout", fmt.Sprint(timeout))
	}
	if sslMode := extraConfig.SSLMode; sslMode != "" {
		query.Add("sslmode", sslMode)
	}
	if sslCert := extraConfig.SSLCert; sslCert != "" {
		query.Add("sslcert", sslCert)
	}
	if sslKey := extraConfig.SSLKey; sslKey != "" {
		query.Add("sslkey", sslKey)
	}
	if rootCert := extraConfig.SSLRootCert; rootCert != "" {
		query.Add("sslrootcert", rootCert)
	}
	targetUrl.RawQuery = query.Encode()
	return targetUrl.String()
}

func contextLogger(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	logger := zerolog.Ctx(ctx)
	var event *zerolog.Event
	switch level {
	case tracelog.LogLevelError:
		event = logger.Error()
	case tracelog.LogLevelWarn:
		event = logger.Warn()
	case tracelog.LogLevelInfo:
		event = logger.Info()
	case tracelog.LogLevelDebug:
		event = logger.Debug()
	case tracelog.LogLevelTrace:
		event = logger.Trace()
	default:
		return
	}
	event.Any("data", data).Msg(msg)
}

func retryConnect(logger *zerolog.Logger, config *pgxpool.Config, outputPool **pgxpool.Pool) retry.RetryFunc {
	return func(ctx context.Context) error {
		pool, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed connecting to PostgreSQL, retying...")
			return retry.RetryableError(err)
		}
		if err := pool.Ping(ctx); err != nil {
			logger.Warn().Err(err).Msg("Failed connecting to PostgreSQL, retying...")
			return retry.RetryableError(err)
		}
		*outputPool = pool
		return nil
	}
}
