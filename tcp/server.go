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
	"github.com/thanhminhmr/go-common/log"

	"github.com/thanhminhmr/go-exception"
)

type ConnectionHandler[Connection net.Conn] func(ctx context.Context, conn Connection) error

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
	return func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner, error) {
		logger := log.IntoCtx(ctx).With("port", config.Port)
		// create listener
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(config.Port)})
		if err != nil {
			logger.Error("Failed to listen", "error", err)
			return nil, nil, exception.String("TcpServer: Failed to listen").AddCause(err)
		}
		logger.Info("Listener started")
		// create wait
		server := tcpServer{
			config:   config,
			handler:  handler,
			listener: listener,
		}
		return server.run, server.cleanUp, nil
	}
}

type tcpServer struct {
	config   *ServerConfig
	handler  ConnectionHandler[*net.TCPConn]
	listener *net.TCPListener
	wait     sync.WaitGroup
}

func (s *tcpServer) run(ctx context.Context, shutdown context.CancelFunc) {
	logger := log.IntoCtx(ctx).With("port", s.config.Port)
	// create semaphore as a concurrent connection limiter
	semaphore := make(chan struct{}, s.config.ConcurrentConnections)
	for {
		// acquiring a slot in the semaphore, blocking while full
		select {
		case <-ctx.Done():
			logger.Info("Stop accepting connection")
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
				logger.Error("Failed to accept connection", "error", err)
				if s.config.ShutdownOnError {
					shutdown()
				}
			}
			return
		}
	}
}

func (s *tcpServer) execute(ctx context.Context, connection *net.TCPConn) {
	logger := log.IntoCtx(ctx).With("connection_id", rand.Text())
	if s.config.TracePerConnection {
		traceLogger := logger.With(
			"remote_address", connection.RemoteAddr(),
			"local_address", connection.LocalAddr(),
		)
		traceLogger.Debug("Start handling connection")
		defer traceLogger.Debug("Finish handling connection")
	}
	defer exception.Recover(func(recovered exception.Exception) {
		logger.Error("Panic while handling connection", "recovered", recovered)
	})
	defer func() {
		if err := connection.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			logger.Error("Failed to close connection", "error", err)
		}
	}()
	if err := s.handler(logger, connection); err != nil {
		logger.Error("Error handling connection", "error", err)
	}
}

func (s *tcpServer) cleanUp(ctx context.Context) {
	defer s.wait.Wait()
	logger := log.IntoCtx(ctx).With("port", s.config.Port)
	// stop the listener
	logger.Info("Stopping listener")
	if err := s.listener.Close(); err != nil {
		logger.Error("Failed to close listener", "error", err)
	}
	logger.Info("Listener stopped")
}
