/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func ShutdownOnSignal(_ context.Context) (Runner, Cleaner) {
	return func(ctx context.Context, shutdown context.CancelFunc) {
		// register exit signal
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		select {
		case received := <-signals:
			Logger(ctx).Info().Stringer("received", received).Msg("Receive signal, shutting down...")
			shutdown()
		case <-ctx.Done():
		}
	}, nil
}
