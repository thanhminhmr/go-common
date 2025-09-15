package http

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/thanhminhmr/go-common/configuration"
	"github.com/thanhminhmr/go-common/exception"
	"github.com/thanhminhmr/go-common/log"
	"go.uber.org/fx"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

type ServerConfig struct {
	Port uint16 `env:"HTTP_SERVER_PORT" validate:"required"`
}

type ServerExtraConfig struct {
	ReadHeaderTimeout uint32 `env:"HTTP_SERVER_READ_HEADER_TIMEOUT" validate:"min=0,max=60"`
	IdleTimeout       uint32 `env:"HTTP_SERVER_IDLE_TIMEOUT" validate:"min=0,max=3600"`
	MaxHeaderBytes    uint32 `env:"HTTP_SERVER_MAX_HEADER_BYTES" validate:"min=0,max=65536"`
}

func init() {
	configuration.SetDefault("HTTP_SERVER_READ_HEADER_TIMEOUT", "5")
	configuration.SetDefault("HTTP_SERVER_IDLE_TIMEOUT", "60")
	configuration.SetDefault("HTTP_SERVER_MAX_HEADER_BYTES", "4096")
}

func NewServer(
	logger *zerolog.Logger,
	lifecycle fx.Lifecycle,
	config *ServerConfig,
	extraConfig *ServerExtraConfig,
) chi.Router {
	// create route
	router := chi.NewRouter()
	// create the http server
	server := httpServer{
		logger: logger,
		router: router,
		server: http.Server{
			Addr:              ":" + strconv.FormatUint(uint64(config.Port), 10),
			Handler:           router,
			ReadHeaderTimeout: time.Duration(extraConfig.ReadHeaderTimeout) * time.Second,
			IdleTimeout:       time.Duration(extraConfig.IdleTimeout) * time.Second,
			MaxHeaderBytes:    int(extraConfig.MaxHeaderBytes),
		},
	}
	// set a sane default middleware stack
	router.Use(
		server.log,
		middleware.StripSlashes,
	)
	// add to lifecycle
	lifecycle.Append(fx.Hook{
		OnStart: server.onStart,
		OnStop:  server.onStop,
	})
	return router
}

type httpServer struct {
	logger *zerolog.Logger
	router *chi.Mux
	server http.Server
}

func (s *httpServer) onStart(_ context.Context) error {
	// dump all routes
	s.logger.Info().Msg("Listing all routes...")
	if err := chi.Walk(s.router, s.dumpRoutes); err != nil {
		s.logger.Error().Err(err).Msg("Error walking routes")
		return err
	}
	s.logger.Info().Msg("Listed all routes")
	// start the server
	go s.serve()
	return nil
}

func (s *httpServer) serve() {
	s.logger.Info().Str("addr", s.server.Addr).Msgf("Start serving")
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error().Err(err).Msg("Shutdown with error")
	}
}

func (s *httpServer) onStop(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down...")
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Shutdown with error")
		return err
	}
	s.logger.Info().Msg("Shutdown complete")
	return nil
}

func (s *httpServer) dumpRoutes(
	method string,
	route string,
	handler http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) error {
	s.logger.Info().
		Stringer("handler", log.Func(handler)).
		Array("middlewares", log.Funcs(middlewares)).
		Msgf("Route: %s %s", method, route)
	return nil
}

func (s *httpServer) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		logger := s.logger.With().Str("request_id", fmt.Sprintf("%016x", rand.Uint64())).Logger()
		// log request and response
		logger.Info().
			Str("method", request.Method).
			Stringer("url", request.URL).
			Msg("Request")
		start := time.Now()
		wrappedWriter := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)
		defer func(start time.Time, wrappedWriter middleware.WrapResponseWriter) {
			duration := time.Since(start)
			logger.Info().
				Int("status", wrappedWriter.Status()).
				Int("bytes", wrappedWriter.BytesWritten()).
				Dur("duration", duration).
				Msg("Response")
		}(start, wrappedWriter)
		// recover any panic
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error().
					Any("recovered", recovered).
					Array("stack", exception.StackTrace(1)).
					Msg("Recovered from panic")
				// response with 500 Internal Server Error
				if request.Header.Get("Connection") != "Upgrade" {
					wrappedWriter.WriteHeader(http.StatusInternalServerError)
				}
			}
		}()
		// call the next handler
		next.ServeHTTP(wrappedWriter, request.WithContext(logger.WithContext(request.Context())))
	})
}
