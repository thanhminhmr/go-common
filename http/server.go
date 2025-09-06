package http

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/thanhminhmr/go-common/log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func NewServer(
	logger *zerolog.Logger,
	lifecycle fx.Lifecycle,
	config *ServerConfig,
	extraConfig *ServerExtraConfig,
) chi.Router {
	// create router
	router := chi.NewRouter()

	// set a sane default middleware stack
	router.Use(
		logInjector(logger),
		middleware.Recoverer,
		middleware.StripSlashes,
		middleware.NoCache,
	)

	// create the http server
	server := &http.Server{
		Addr:              ":" + strconv.FormatUint(uint64(config.Port), 10),
		Handler:           router,
		ReadHeaderTimeout: time.Duration(extraConfig.ReadHeaderTimeout) * time.Second,
		IdleTimeout:       time.Duration(extraConfig.IdleTimeout) * time.Second,
		MaxHeaderBytes:    int(extraConfig.MaxHeaderBytes),
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// dump all routes
			logger.Info().Msg("Listing all routes...")
			err := chi.Walk(
				router,
				func(
					method string,
					route string,
					handler http.Handler,
					middlewares ...func(http.Handler) http.Handler,
				) error {
					logger.Info().
						Stringer("handler", log.Func(handler)).
						Array("middlewares", log.Funcs(middlewares)).
						Msgf("Route: %s %s", method, route)
					return nil
				},
			)
			if err != nil {
				logger.Error().Err(err).Msg("Error walking routes")
				return err
			}
			logger.Info().Msg("Listed all routes")
			// start the server
			go func() {
				logger.Info().Msgf("HTTP server starting on port %s", server.Addr)
				err := server.ListenAndServe()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error().Err(err).Msg("HTTP server shutdown with error")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// shutdown server
			logger.Info().Msg("HTTP server shutting down...")
			if err := server.Shutdown(context.Background()); err != nil {
				logger.Error().Err(err).Msg("HTTP server shutdown with error")
				return err
			}
			logger.Info().Msg("HTTP server shutdown complete")
			return nil
		},
	})
	return router
}

func logInjector(logger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestLogger := logger.With().Str("request_id", fmt.Sprintf("%016x%016x", rand.Uint64(), rand.Uint64())).Logger()
			// log request
			requestLogger.Info().Str("method", request.Method).Str("path", request.URL.Path).Msg("Request")
			// log response
			start := time.Now()
			wrappedWriter := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)
			defer func() {
				duration := time.Since(start)
				requestLogger.Info().
					Int("Status", wrappedWriter.Status()).
					Int("bytes", wrappedWriter.BytesWritten()).
					Dur("duration", duration).
					Msg("Response")
			}()
			// call the next handler
			next.ServeHTTP(wrappedWriter, request.WithContext(requestLogger.WithContext(request.Context())))
		})
	}
}
