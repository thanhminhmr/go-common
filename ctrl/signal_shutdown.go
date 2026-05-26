package ctrl

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/thanhminhmr/go-common/log"
)

func ShutdownOnSignal(_ context.Context) (Runner, Cleaner, error) {
	return func(ctx context.Context, shutdown context.CancelFunc) {
		// register exit signal
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		select {
		case received := <-signals:
			log.IntoCtx(ctx).Info("Receive signal, shutting down...", "received", received)
			shutdown()
		case <-ctx.Done():
		}
	}, nil, nil
}
