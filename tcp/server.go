/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package tcp

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"sync"

	"github.com/rs/zerolog"

	"github.com/thanhminhmr/go-common/ctrl"

	"github.com/thanhminhmr/go-exception"
)

// ConnectionHandler handles a single accepted TCP connection.
type ConnectionHandler[Connection net.Conn] = func(ctx context.Context, conn Connection) error

// ServerConfig configures a TCP server created by [NewServer].
type ServerConfig struct {
	Port                  uint16 `cfg:"port" default:"32768" validate:"required"`
	ShutdownOnError       bool   `cfg:"shutdown_on_error" default:"true"`
	TracePerConnection    bool   `cfg:"trace_per_connection"`
	ConcurrentConnections uint32 `cfg:"concurrent_connections" validate:"min=256,max=65536" default:"1024"`
}

// NewServer returns a [ctrl.Starter] that, when started, listens for TCP
// connections on config.Port and dispatches each accepted connection to handler.
// The returned Runner accepts connections concurrently up to
// config.ConcurrentConnections; the returned Cleaner closes the listener and
// waits for in-flight handlers to finish. It panics if the listener cannot be
// created.
func NewServer(config *ServerConfig, handler ConnectionHandler[*net.TCPConn]) ctrl.Starter {
	return func(ctx, _ context.Context) (ctrl.Runner, ctrl.Cleaner) {
		logger := zerolog.Ctx(ctx).With().Uint16("port", config.Port).Logger()
		// create listener
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(config.Port)})
		if err != nil {
			exception.Panic(exception.String("TcpServer: Failed to listen").AddCause(err))
		}
		logger.Info().Msg("Listener started")
		// create wait
		server := tcpServer{
			config:   config,
			handler:  handler,
			listener: listener,
		}
		return server.run, server.cleanUp
	}
}

type tcpServer struct {
	config   *ServerConfig
	handler  ConnectionHandler[*net.TCPConn]
	listener *net.TCPListener
	wait     sync.WaitGroup
}

func (s *tcpServer) run(ctx context.Context, shutdown context.CancelFunc) {
	logger := zerolog.Ctx(ctx).With().Uint16("port", s.config.Port).Logger()
	ctx = logger.WithContext(ctx)
	// create semaphore as a concurrent connection limiter
	semaphore := make(chan struct{}, s.config.ConcurrentConnections)
	for {
		// acquiring a slot in the semaphore, blocking while full
		select {
		case <-ctx.Done():
			logger.Info().Msg("Stop accepting connection")
			return
		case semaphore <- struct{}{}:
		}
		// accept a connection and execute the connection handler
		if connection, err := s.listener.AcceptTCP(); err == nil {
			s.wait.Go(func() {
				s.execute(ctx, connection)
				<-semaphore
			})
		} else {
			// only if the server is not closed already
			if ctx.Err() == nil {
				logger.Error().Err(err).Msg("Failed to accept connection")
				if s.config.ShutdownOnError {
					shutdown()
				}
			}
			return
		}
	}
}

func (s *tcpServer) execute(ctx context.Context, connection *net.TCPConn) {
	logger := zerolog.Ctx(ctx).With().Str("connection_id", rand.Text()).Logger()
	ctx = logger.WithContext(ctx)
	if s.config.TracePerConnection {
		traceLogger := logger.With().
			Stringer("remote_address", connection.RemoteAddr()).
			Stringer("local_address", connection.LocalAddr()).
			Logger()
		traceLogger.Debug().Msg("Start handling connection")
		defer traceLogger.Debug().Msg("Finish handling connection")
	}
	defer exception.Recover(func(recovered exception.Exception) {
		logger.Error().AnErr("recovered", recovered).Msg("Panic while handling connection")
	})
	defer func() {
		if err := connection.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error().Err(err).Msg("Failed to close connection")
		}
	}()
	if err := s.handler(ctx, connection); err != nil {
		logger.Error().Err(err).Msg("Error handling connection")
	}
}

func (s *tcpServer) cleanUp(ctx context.Context) {
	defer s.wait.Wait()
	logger := zerolog.Ctx(ctx).With().Uint16("port", s.config.Port).Logger()
	logger.Info().Msg("Stopping listener")
	if err := s.listener.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close listener")
	}
	logger.Info().Msg("Listener stopped")
}
