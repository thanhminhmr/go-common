package tcp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Address     netip.AddrPort `env:"TCP_SERVER_ADDRESS" validate:"required"`
	Concurrency uint16         `env:"TCP_SERVER_CONCURRENCY" validate:"min=1,max=256"`
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
		logger:   logger,
		shutdown: shutdown,
		config:   config,
		handler:  handler,
		ctx:      serverCtx,
		cancel:   cancel,
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
	listener  atomic.Pointer[net.TCPListener]
	waitGroup sync.WaitGroup
}

func (s *tcpServer) onStart(_ context.Context) error {
	// create listener
	listener, err := net.ListenTCP("tcp", net.TCPAddrFromAddrPort(s.config.Address))
	if err != nil {
		s.logger.Error().Err(err).Stringer("addr", s.config.Address).Msg("Failed to listen")
		return err
	}
	s.listener.Store(listener)
	// start workers
	s.waitGroup.Add(int(s.config.Concurrency))
	s.logger.Info().Stringer("addr", s.config.Address).Msg("Start listening")
	for range s.config.Concurrency {
		go s.worker()
	}
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
	defer s.waitGroup.Done()
	listener := s.listener.Load()
	for {
		if connection, err := listener.AcceptTCP(); err == nil {
			s.execute(connection)
			continue
		} else if s.listener.Load() != nil {
			s.logger.Error().Err(err).Msg("Failed to accept connection")
		}
		break
	}
}

func (s *tcpServer) execute(connection *net.TCPConn) {
	logger := s.logger.With().Str("connection_id", fmt.Sprintf("%016x", rand.Uint64())).Logger()
	logger.Info().
		Stringer("remote_address", connection.RemoteAddr()).
		Stringer("local_address", connection.LocalAddr()).
		Msg("Start handling connection")
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error().Any("recovered", recovered).Msg("Panic while handling connection")
		}
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
	s.logger.Info().Stringer("addr", s.config.Address).Msg("Stop listening")
	go func() {
		s.waitGroup.Wait()
		s.cancel()
	}()
	select {
	case <-s.ctx.Done():
	case <-ctx.Done():
		s.cancel()
	}
	return nil
}
