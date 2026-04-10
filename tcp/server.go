/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package tcp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"

	"github.com/rs/zerolog"
	"github.com/thanhminhmr/go-exception"
	"go.uber.org/fx"
)

type ConnectionHandler[Connection net.Conn] func(ctx context.Context, conn Connection) error

type ServerConfig struct {
	Port                  uint16 `env:"TCP_SERVER_PORT" validate:"required"`
	ShutdownOnError       bool   `env:"TCP_SERVER_SHUTDOWN_ON_ERROR" default:"true"`
	TracePerConnection    bool   `env:"TCP_SERVER_TRACE_PER_CONNECTION"`
	ConcurrentConnections uint32 `env:"TCP_SERVER_CONCURRENT_CONNECTIONS" validate:"min=256,max=65536" default:"1024"`
}

type tcpServer struct {
	ctx       context.Context
	cancel    context.CancelFunc
	shutdown  fx.Shutdowner
	logger    zerolog.Logger
	config    *ServerConfig
	handler   ConnectionHandler[*net.TCPConn]
	waitGroup sync.WaitGroup
}

func NewServer(
	lifecycle fx.Lifecycle,
	shutdown fx.Shutdowner,
	logger *zerolog.Logger,
	config *ServerConfig,
	handler ConnectionHandler[*net.TCPConn],
) {
	ctx, cancel := context.WithCancel(context.Background())
	server := tcpServer{
		ctx:      ctx,
		cancel:   cancel,
		shutdown: shutdown,
		logger:   logger.With().Uint16("port", config.Port).Logger(),
		config:   config,
		handler:  handler,
	}
	lifecycle.Append(fx.Hook{
		OnStart: server.onStart,
		OnStop:  server.onStop,
	})
}

func (s *tcpServer) onStart(context.Context) error {
	// create listener
	//goland:noinspection GoResourceLeak
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(s.config.Port)})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to listen")
		return err
	}
	// start workers
	s.logger.Info().Msg("Listener started")
	go s.worker(listener)
	return nil
}

func (s *tcpServer) worker(listener *net.TCPListener) {
	s.waitGroup.Add(1)
	// register listener closer
	closer := sync.OnceFunc(func() {
		if err := listener.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Failed to close listener")
		}
		s.logger.Info().Msg("Listener stopped")
		s.waitGroup.Done()
	})
	defer closer()
	context.AfterFunc(s.ctx, closer)
	// create semaphore as a concurrent connection limiter
	semaphore := make(chan struct{}, s.config.ConcurrentConnections)
	for {
		// acquiring a slot in the semaphore, blocking while full
		select {
		case <-s.ctx.Done():
			s.logger.Info().Msg("Stop accepting connection")
			return
		case semaphore <- struct{}{}:
		}
		// accept a connection and execute the connection handler
		if connection, err := listener.AcceptTCP(); err == nil {
			s.waitGroup.Go(func() {
				s.execute(connection)
				<-semaphore
			})
			continue
		} else if s.ctx.Err() == nil {
			// only if the server is not closed already
			s.logger.Error().Err(err).Msg("Failed to accept connection")
			if s.config.ShutdownOnError {
				if err := s.shutdown.Shutdown(); err != nil {
					s.logger.Error().Err(err).Msg("Failed to send shutdown signal")
				} else {
					s.logger.Info().Err(err).Msg("Sent shutdown signal")
				}
			}
		}
		break
	}
}

func (s *tcpServer) execute(connection *net.TCPConn) {
	logger := s.logger.With().Str("connection_id", fmt.Sprintf("%016x", rand.Uint64())).Logger()
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
	if err := s.handler(logger.WithContext(s.ctx), connection); err != nil {
		logger.Error().Err(err).Msg("Error handling connection")
	}
}

func (s *tcpServer) onStop(ctx context.Context) error {
	s.logger.Info().Msg("Stopping listener")
	s.cancel()
	// waiting for connection to finish
	done := make(chan struct{})
	go func(done chan<- struct{}) {
		s.waitGroup.Wait()
		close(done)
	}(done)
	// ... or timeout/cancel from global context
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
