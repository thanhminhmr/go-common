package tcp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"github.com/thanhminhmr/go-exception"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Port               uint16 `env:"TCP_SERVER_PORT" validate:"required"`
	ShutdownOnError    bool   `env:"TCP_SERVER_SHUTDOWN_ON_ERROR"`
	TracePerConnection bool   `env:"TCP_SERVER_TRACE_PER_CONNECTION"`
}

type ServerHandler interface {
	Handle(ctx context.Context, conn *net.TCPConn) error
}

type ServerHandlerFunc func(ctx context.Context, conn *net.TCPConn) error

func (f ServerHandlerFunc) Handle(ctx context.Context, conn *net.TCPConn) error {
	return f(ctx, conn)
}

func NewServer(
	ctx context.Context,
	lifecycle fx.Lifecycle,
	shutdown fx.Shutdowner,
	config *ServerConfig,
	handler ServerHandler,
) {
	server := tcpServer{
		ctx:       ctx,
		shutdown:  shutdown,
		config:    config,
		handler:   handler,
		semaphore: make(chan struct{}, 1024),
	}
	lifecycle.Append(fx.Hook{
		OnStart: server.onStart,
		OnStop:  server.onStop,
	})
}

type tcpServer struct {
	ctx       context.Context
	shutdown  fx.Shutdowner
	config    *ServerConfig
	handler   ServerHandler
	semaphore chan struct{}
	listener  atomic.Pointer[net.TCPListener]
	waitGroup sync.WaitGroup
}

func (s *tcpServer) onStart(context.Context) error {
	logger := zerolog.Ctx(s.ctx)
	// create listener
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(s.config.Port)})
	if err != nil {
		logger.Error().Err(err).Uint16("port", s.config.Port).Msg("Failed to listen")
		return err
	}
	s.listener.Store(listener)
	// start workers
	logger.Info().Uint16("port", s.config.Port).Msg("Start listening")
	go s.worker()
	return nil
}

func (s *tcpServer) halt(unexpected bool) {
	if listener := s.listener.Swap(nil); listener != nil {
		logger := zerolog.Ctx(s.ctx)
		if err := listener.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close listener")
		}
		// shutdown with exit code if worker failed unexpectedly
		if unexpected && s.config.ShutdownOnError {
			if err := s.shutdown.Shutdown(fx.ExitCode(1)); err != nil {
				logger.Error().Err(err).Msg("Failed to send shutdown signal")
			}
		}
	}
}

func (s *tcpServer) worker() {
	defer s.halt(true)
	logger := zerolog.Ctx(s.ctx)
	listener := s.listener.Load()
	for {
		// acquiring a slot in the semaphore, blocking while full
		select {
		case <-s.ctx.Done():
			logger.Error().Err(s.ctx.Err()).Msg("Stop accepting connection")
			return
		case s.semaphore <- struct{}{}:
		}
		// accept a connection and execute the connection handler
		if connection, err := listener.AcceptTCP(); err == nil {
			go s.execute(connection)
			continue
		} else if s.listener.Load() != nil {
			logger.Error().Err(err).Msg("Failed to accept connection")
		}
		break
	}
}

func (s *tcpServer) execute(connection *net.TCPConn) {
	s.waitGroup.Add(1)
	logger := zerolog.Ctx(s.ctx).With().Str("connection_id", fmt.Sprintf("%016x", rand.Uint64())).Logger()
	if s.config.TracePerConnection {
		logger.Trace().
			Stringer("remote_address", connection.RemoteAddr()).
			Stringer("local_address", connection.LocalAddr()).
			Msg("Start handling connection")
	}
	defer func() {
		if recovered := exception.Recover(recover()); recovered != nil {
			logger.Error().Any("recovered", recovered).Msg("Panic while handling connection")
		}
		s.waitGroup.Done()
		<-s.semaphore
		if s.config.TracePerConnection {
			logger.Trace().
				Stringer("remote_address", connection.RemoteAddr()).
				Stringer("local_address", connection.LocalAddr()).
				Msg("Finish handling connection")
		}
	}()
	defer func() {
		if err := connection.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close connection")
		}
	}()
	if err := s.handler.Handle(logger.WithContext(s.ctx), connection); err != nil {
		logger.Error().Err(err).Msg("Error handling connection")
	}
}

func (s *tcpServer) onStop(ctx context.Context) error {
	s.halt(false)
	zerolog.Ctx(s.ctx).Info().Uint16("port", s.config.Port).Msg("Stop listening")
	// waiting for connection to finish
	done := make(chan struct{})
	go func(done chan<- struct{}) {
		s.waitGroup.Wait()
		close(done)
	}(done)
	// ... or timeout/cancel from global/local context
	select {
	case <-s.ctx.Done():
	case <-ctx.Done():
	case <-done:
	}
	return nil
}
