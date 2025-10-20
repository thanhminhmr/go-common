package tcp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Port            uint16 `env:"TCP_SERVER_PORT" validate:"required"`
	ShutdownTimeout uint   `env:"TCP_SERVER_SHUTDOWN_TIMEOUT" validate:"required,max=10"`
}

type ServerHandler interface {
	Handle(ctx context.Context, conn *net.TCPConn) error
}

type ServerHandlerFunc func(ctx context.Context, conn *net.TCPConn) error

func (f ServerHandlerFunc) Handle(ctx context.Context, conn *net.TCPConn) error {
	return f(ctx, conn)
}

func NewServer(
	logger *zerolog.Logger,
	lifecycle fx.Lifecycle,
	shutdown fx.Shutdowner,
	config *ServerConfig,
	handler ServerHandler,
) {
	serverCtx, cancel := context.WithCancel(context.Background())
	server := tcpServer{
		logger:    logger,
		shutdown:  shutdown,
		config:    config,
		handler:   handler,
		ctx:       serverCtx,
		cancel:    cancel,
		semaphore: make(chan struct{}, 1024),
	}
	lifecycle.Append(fx.Hook{
		OnStart: server.onStart,
		OnStop:  server.onStop,
	})
}

type tcpServer struct {
	logger    *zerolog.Logger
	shutdown  fx.Shutdowner
	config    *ServerConfig
	handler   ServerHandler
	ctx       context.Context
	cancel    context.CancelFunc
	semaphore chan struct{}
	listener  atomic.Pointer[net.TCPListener]
	waitGroup sync.WaitGroup
}

func (s *tcpServer) onStart(_ context.Context) error {
	// create listener
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(s.config.Port)})
	if err != nil {
		s.logger.Error().Err(err).Uint16("port", s.config.Port).Msg("Failed to listen")
		return err
	}
	s.listener.Store(listener)
	// start workers
	s.logger.Info().Uint16("port", s.config.Port).Msg("Start listening")
	go s.worker()
	return nil
}

func (s *tcpServer) halt(unexpected bool) {
	if listener := s.listener.Swap(nil); listener != nil {
		if err := listener.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Failed to close listener")
		}
		// set exit code if worker failed unexpectedly
		var opts []fx.ShutdownOption
		if unexpected {
			opts = []fx.ShutdownOption{fx.ExitCode(1)}
		}
		_ = s.shutdown.Shutdown(opts...)
	}
}

func (s *tcpServer) worker() {
	defer s.halt(true)
	listener := s.listener.Load()
	for {
		// acquiring a slot in the semaphore, blocking while full
		select {
		case <-s.ctx.Done():
			s.logger.Error().Err(s.ctx.Err()).Msg("Stop accepting connection")
			return
		case s.semaphore <- struct{}{}:
		}
		// accept a connection and execute the connection handler
		if connection, err := listener.AcceptTCP(); err == nil {
			go s.execute(connection)
			continue
		} else if s.listener.Load() != nil {
			s.logger.Error().Err(err).Msg("Failed to accept connection")
		}
		break
	}
}

func (s *tcpServer) execute(connection *net.TCPConn) {
	s.waitGroup.Add(1)
	logger := s.logger.With().Str("connection_id", fmt.Sprintf("%016x", rand.Uint64())).Logger()
	logger.Info().
		Stringer("remote_address", connection.RemoteAddr()).
		Stringer("local_address", connection.LocalAddr()).
		Msg("Start handling connection")
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error().Any("recovered", recovered).Msg("Panic while handling connection")
		}
		s.waitGroup.Done()
		<-s.semaphore
		logger.Info().
			Stringer("remote_address", connection.RemoteAddr()).
			Stringer("local_address", connection.LocalAddr()).
			Msg("Finish handling connection")
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
	s.logger.Info().Uint16("port", s.config.Port).Msg("Stop listening")
	go func() {
		s.waitGroup.Wait()
		s.cancel()
	}()
	select {
	case <-s.ctx.Done():
	case <-ctx.Done():
	case <-time.After(time.Duration(s.config.ShutdownTimeout) * time.Second):
	}
	s.cancel()
	return nil
}
