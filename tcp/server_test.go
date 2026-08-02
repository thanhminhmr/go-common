/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package tcp_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thanhminhmr/go-common/tcp"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

func testContext() context.Context {
	return zerolog.New(io.Discard).WithContext(context.Background())
}

func dial(t *testing.T, port int) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	return conn
}

func waitRecv(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal(msg)
	}
}

func assertNoRecv(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal(msg)
	default:
	}
}

// silentShutdown returns a CancelFunc-like function that records a call without
// blocking. It is used in place of the real shutdown to detect whether the
// server requested a shutdown.
func silentShutdown(recorded chan<- struct{}) context.CancelFunc {
	return func() {
		select {
		case recorded <- struct{}{}:
		default:
		}
	}
}

// startServer creates a TCP server on a free port, starts its runner, and
// returns the port and a cleanup function that cancels the run context and
// calls the cleaner. config.Port is overwritten with a free port.
func startServer(
	t *testing.T,
	config *tcp.ServerConfig,
	handler tcp.ConnectionHandler[*net.TCPConn],
) (port int, cleanup func()) {
	t.Helper()
	ctx := testContext()
	port = freePort(t)
	config.Port = uint16(port)
	starter := tcp.NewServer(config, handler)
	runner, cleaner := starter(ctx, ctx)
	runCtx, cancel := context.WithCancel(ctx)
	go runner(runCtx, cancel)
	return port, func() {
		cancel()
		cleaner(ctx)
	}
}

// startServerWithShutdown is like [startServer] but passes a [silentShutdown]
// to the runner instead of the real cancel func. It returns the server context,
// the shutdown-signal channel, the run-context cancel func, the cleaner, and a
// channel closed when the runner returns. It is used by tests that observe
// whether the server requested a shutdown.
func startServerWithShutdown(
	t *testing.T,
	config *tcp.ServerConfig,
	handler tcp.ConnectionHandler[*net.TCPConn],
) (ctx context.Context, shutdownCalled <-chan struct{}, cancel context.CancelFunc, cleaner func(context.Context), done <-chan struct{}) {
	t.Helper()
	ctx = testContext()
	port := freePort(t)
	config.Port = uint16(port)
	starter := tcp.NewServer(config, handler)
	runner, clean := starter(ctx, ctx)
	runCtx, cancelFn := context.WithCancel(ctx)
	shutdownCh := make(chan struct{}, 1)
	doneCh := make(chan struct{})
	go func() {
		runner(runCtx, silentShutdown(shutdownCh))
		close(doneCh)
	}()
	return ctx, shutdownCh, cancelFn, clean, doneCh
}

func TestNewServer_ReturnsStarter(t *testing.T) {
	ctx := testContext()
	port := freePort(t)
	starter := tcp.NewServer(
		&tcp.ServerConfig{Port: uint16(port), ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error { return nil },
	)
	require.NotNil(t, starter)
	runner, cleaner := starter(ctx, ctx)
	require.NotNil(t, runner)
	require.NotNil(t, cleaner)
	cleaner(ctx)
}

func TestNewServer_ListenFailure_Panics(t *testing.T) {
	ctx := testContext()
	// Occupy a port so that NewServer cannot listen on it.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func(l net.Listener) { _ = l.Close() }(l)
	port := l.Addr().(*net.TCPAddr).Port

	starter := tcp.NewServer(
		&tcp.ServerConfig{Port: uint16(port), ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error { return nil },
	)
	assert.Panics(t, func() { starter(ctx, ctx) })
}

func TestServer_AcceptsConnection(t *testing.T) {
	handled := make(chan struct{}, 1)
	port, cleanup := startServer(t,
		&tcp.ServerConfig{ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error {
			handled <- struct{}{}
			return nil
		},
	)
	defer cleanup()

	conn := dial(t, port)
	defer func(conn net.Conn) { _ = conn.Close() }(conn)
	waitRecv(t, handled, "handler not called")
}

func TestServer_HandlerError(t *testing.T) {
	calls := make(chan struct{}, 2)
	port, cleanup := startServer(t,
		&tcp.ServerConfig{ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error {
			calls <- struct{}{}
			return errors.New("boom")
		},
	)
	defer cleanup()

	c1 := dial(t, port)
	waitRecv(t, calls, "first handler not called")
	_ = c1.Close()
	c2 := dial(t, port)
	waitRecv(t, calls, "second handler not called")
	_ = c2.Close()
}

func TestServer_HandlerPanic(t *testing.T) {
	calls := make(chan struct{}, 2)
	port, cleanup := startServer(t,
		&tcp.ServerConfig{ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error {
			calls <- struct{}{}
			panic("boom")
		},
	)
	defer cleanup()

	c1 := dial(t, port)
	waitRecv(t, calls, "first handler not called")
	_ = c1.Close()
	c2 := dial(t, port)
	waitRecv(t, calls, "second handler not called")
	_ = c2.Close()
}

func TestServer_HandlerClosesConnection(t *testing.T) {
	handled := make(chan struct{}, 1)
	port, cleanup := startServer(t,
		&tcp.ServerConfig{ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error {
			// Handler closes first; deferred Close returns net.ErrClosed.
			require.NoError(t, conn.Close())
			handled <- struct{}{}
			return nil
		},
	)
	defer cleanup()

	conn := dial(t, port)
	defer func(conn net.Conn) { _ = conn.Close() }(conn)
	waitRecv(t, handled, "handler not called")
}

func TestServer_TracePerConnection(t *testing.T) {
	handled := make(chan struct{}, 1)
	port, cleanup := startServer(t,
		&tcp.ServerConfig{ConcurrentConnections: 256, TracePerConnection: true},
		func(ctx context.Context, conn *net.TCPConn) error {
			handled <- struct{}{}
			return nil
		},
	)
	defer cleanup()

	conn := dial(t, port)
	defer func(conn net.Conn) { _ = conn.Close() }(conn)
	waitRecv(t, handled, "handler not called")
}

func TestServer_Run_StopsOnContextCancel(t *testing.T) {
	ctx := testContext()
	port := freePort(t)
	release := make(chan struct{})
	started := make(chan struct{}, 256)
	starter := tcp.NewServer(
		&tcp.ServerConfig{Port: uint16(port), ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error {
			started <- struct{}{}
			<-release
			return nil
		},
	)
	runner, cleaner := starter(ctx, ctx)
	runCtx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})
	go func() {
		runner(runCtx, cancel)
		close(done)
	}()
	defer func() {
		cancel()
		close(release)
		cleaner(ctx)
	}()

	// Fill the semaphore so the next select blocks on ctx.Done().
	var conns []net.Conn
	for i := 0; i < 256; i++ {
		conns = append(conns, dial(t, port))
	}
	for i := 0; i < 256; i++ {
		waitRecv(t, started, "handler not started")
	}

	// Semaphore is now full; the accept loop is blocked acquiring a slot.
	cancel()
	waitRecv(t, done, "runner did not stop on context cancel")

	for _, c := range conns {
		if err := c.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("unexpected close error: %v", err)
		}
	}
}

func TestServer_Run_AcceptError_ShutdownOnError(t *testing.T) {
	ctx, shutdownCalled, cancel, cleaner, done := startServerWithShutdown(t,
		&tcp.ServerConfig{ConcurrentConnections: 256, ShutdownOnError: true},
		func(ctx context.Context, conn *net.TCPConn) error { return nil },
	)
	defer cancel()

	// Closing the listener causes AcceptTCP to fail with ctx.Err() == nil.
	cleaner(ctx)

	waitRecv(t, shutdownCalled, "shutdown not called")
	waitRecv(t, done, "runner did not stop")
}

func TestServer_Run_AcceptError_NoShutdownOnError(t *testing.T) {
	ctx, shutdownCalled, cancel, cleaner, done := startServerWithShutdown(t,
		&tcp.ServerConfig{ConcurrentConnections: 256, ShutdownOnError: false},
		func(ctx context.Context, conn *net.TCPConn) error { return nil },
	)
	defer cancel()

	cleaner(ctx)

	waitRecv(t, done, "runner did not stop")
	assertNoRecv(t, shutdownCalled, "shutdown should not be called when ShutdownOnError is false")
}

func TestServer_Run_AcceptError_ContextCanceled(t *testing.T) {
	ctx, shutdownCalled, cancel, cleaner, done := startServerWithShutdown(t,
		&tcp.ServerConfig{ConcurrentConnections: 256, ShutdownOnError: true},
		func(ctx context.Context, conn *net.TCPConn) error { return nil },
	)
	// Give the runner time to enter AcceptTCP so that closing the listener
	// (rather than ctx.Done in the select) drives the return path.
	time.Sleep(50 * time.Millisecond)

	// Cancel first, then close the listener: AcceptTCP fails with ctx.Err() != nil.
	cancel()
	cleaner(ctx)

	waitRecv(t, done, "runner did not stop")
	assertNoRecv(t, shutdownCalled, "shutdown should not be called when context is already canceled")
}

func TestServer_CleanUp_WaitsForHandlers(t *testing.T) {
	ctx := testContext()
	port := freePort(t)
	handlerStarted := make(chan struct{}, 1)
	releaseHandler := make(chan struct{})
	handlerDone := make(chan struct{}, 1)
	starter := tcp.NewServer(
		&tcp.ServerConfig{Port: uint16(port), ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error {
			handlerStarted <- struct{}{}
			<-releaseHandler
			handlerDone <- struct{}{}
			return nil
		},
	)
	runner, cleaner := starter(ctx, ctx)
	runCtx, cancel := context.WithCancel(ctx)
	go runner(runCtx, cancel)

	conn := dial(t, port)
	defer func(conn net.Conn) { _ = conn.Close() }(conn)
	waitRecv(t, handlerStarted, "handler not started")

	cancel() // stop accepting new connections

	cleanerDone := make(chan struct{})
	go func() {
		cleaner(ctx)
		close(cleanerDone)
	}()

	// Cleaner closes the listener then waits for the in-flight handler.
	assertNoRecv(t, cleanerDone, "cleaner should wait for in-flight handler")

	close(releaseHandler)
	waitRecv(t, handlerDone, "handler did not complete")
	waitRecv(t, cleanerDone, "cleaner did not return after handler completed")
}

func TestServer_CleanUp_DoubleClose(t *testing.T) {
	ctx := testContext()
	port := freePort(t)
	starter := tcp.NewServer(
		&tcp.ServerConfig{Port: uint16(port), ConcurrentConnections: 256},
		func(ctx context.Context, conn *net.TCPConn) error { return nil },
	)
	_, cleaner := starter(ctx, ctx)

	cleaner(ctx) // first close succeeds
	cleaner(ctx) // second close errors (logged), exercising the error branch
}

func TestServer_ConcurrentConnectionsLimit(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 2)
	port, cleanup := startServer(t,
		&tcp.ServerConfig{ConcurrentConnections: 1},
		func(ctx context.Context, conn *net.TCPConn) error {
			started <- struct{}{}
			<-release
			return nil
		},
	)
	defer cleanup()

	c1 := dial(t, port)
	defer func(c1 net.Conn) { _ = c1.Close() }(c1)
	waitRecv(t, started, "first handler not started")

	c2 := dial(t, port)
	defer func(c2 net.Conn) { _ = c2.Close() }(c2)
	// The second handler must wait: the single semaphore slot is held by c1.
	assertNoRecv(t, started, "second handler should not start while first is in flight")

	close(release)
	waitRecv(t, started, "second handler not started after first released")
}
