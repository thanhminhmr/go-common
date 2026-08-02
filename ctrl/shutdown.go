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

	"github.com/rs/zerolog"
)

// ShutdownOnSignal is a [Starter] that returns a [Runner] listening for
// SIGINT/SIGTERM. On receipt of either signal the runner calls shutdown to
// initiate a graceful controller shutdown. It returns a nil [Cleaner].
func ShutdownOnSignal(_, _ context.Context) (Runner, Cleaner) {
	return func(ctx context.Context, shutdown context.CancelFunc) {
		// register exit signal
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		select {
		case received := <-signals:
			zerolog.Ctx(ctx).Info().Stringer("received", received).Msg("Receive signal, shutting down...")
			shutdown()
		case <-ctx.Done():
		}
	}, nil
}
