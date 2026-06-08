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

	"github.com/thanhminhmr/go-common/ctrl"

	"github.com/thanhminhmr/go-exception"
)

type ConnectionHandler[Connection net.Conn] = func(ctx context.Context, conn Connection) error

type ServerConfig struct {
	Port                  uint16 `env:"TCP_SERVER_PORT" validate:"required"`
	ShutdownOnError       bool   `env:"TCP_SERVER_SHUTDOWN_ON_ERROR" default:"true"`
	TracePerConnection    bool   `env:"TCP_SERVER_TRACE_PER_CONNECTION"`
	ConcurrentConnections uint32 `env:"TCP_SERVER_CONCURRENT_CONNECTIONS" validate:"min=256,max=65536" default:"1024"`
}

func NewServer(
	config *ServerConfig,
	handler ConnectionHandler[*net.TCPConn],
) ctrl.Starter {
	return func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) {
		logCtx := ctrl.LogCtx(ctx).With().Uint16("port", config.Port).Logger()
		// create listener
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(config.Port)})
		if err != nil {
			exception.Panic(exception.String("TcpServer: Failed to listen").AddCause(err))
		}
		logCtx.Info().Msg("Listener started")
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
	logCtx := ctrl.LogCtx(ctx).With().Uint16("port", s.config.Port).Logger()
	// create semaphore as a concurrent connection limiter
	semaphore := make(chan struct{}, s.config.ConcurrentConnections)
	for {
		// acquiring a slot in the semaphore, blocking while full
		select {
		case <-ctx.Done():
			logCtx.Info().Msg("Stop accepting connection")
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
				logCtx.Error().Err(err).Msg("Failed to accept connection")
				if s.config.ShutdownOnError {
					shutdown()
				}
			}
			return
		}
	}
}

func (s *tcpServer) execute(ctx context.Context, connection *net.TCPConn) {
	logCtx := ctrl.LogCtx(ctx).With().Str("connection_id", rand.Text()).Logger()
	if s.config.TracePerConnection {
		traceLogCtx := logCtx.With().
			Stringer("remote_address", connection.RemoteAddr()).
			Stringer("local_address", connection.LocalAddr()).
			Logger()
		traceLogCtx.Debug().Msg("Start handling connection")
		defer traceLogCtx.Debug().Msg("Finish handling connection")
	}
	defer exception.Recover(func(recovered exception.Exception) {
		logCtx.Error().AnErr("recovered", recovered).Msg("Panic while handling connection")
	})
	defer func() {
		if err := connection.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			logCtx.Error().Err(err).Msg("Failed to close connection")
		}
	}()
	if err := s.handler(logCtx, connection); err != nil {
		logCtx.Error().Err(err).Msg("Error handling connection")
	}
}

func (s *tcpServer) cleanUp(ctx context.Context) {
	defer s.wait.Wait()
	logCtx := ctrl.LogCtx(ctx).With().Uint16("port", s.config.Port).Logger()
	// stop the listener
	logCtx.Info().Msg("Stopping listener")
	if err := s.listener.Close(); err != nil {
		logCtx.Error().Err(err).Msg("Failed to close listener")
	}
	logCtx.Info().Msg("Listener stopped")
}
