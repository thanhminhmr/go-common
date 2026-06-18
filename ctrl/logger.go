/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

var logWriter asyncStdoutWriter
var globalLogger zerolog.Logger

func ifValue[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func setupLogger() {
	// setup logger
	zerolog.SetGlobalLevel(config.LoggerMinimumLevel)
	zerolog.TimeFieldFormat = ifValue(config.LoggerUnixTimestamp, zerolog.TimeFormatUnixNano, time.RFC3339Nano)
	// set default logger
	logWriter = asyncStdoutWriter{sent: make(chan []byte, 256), closed: make(chan struct{})}
	globalLogger = zerolog.New(&logWriter).With().Timestamp().Caller().Logger()
	zerolog.DefaultContextLogger = &globalLogger
	// start the log writer
	go logWriter.pollWriter()
}

const bufferCapacity = 4096

var buffers = sync.Pool{New: func() any { return make([]byte, 0, bufferCapacity) }}

type asyncStdoutWriter struct {
	sent   chan []byte
	closed chan struct{}
	lock   sync.RWMutex
	mode   atomic.Bool
}

func (w *asyncStdoutWriter) WriteLevel(_ LogLevel, buffer []byte) (n int, err error) {
	return w.Write(buffer)
}

func (w *asyncStdoutWriter) Write(buffer []byte) (n int, err error) {
	// check if sync mode
	switch {
	case !w.mode.Load():
		// async mode, try to write
		w.lock.RLock()
		// check if sync mode again
		if w.mode.Load() {
			w.lock.RUnlock()
			break
		}
		defer w.lock.RUnlock()
		// async send buffer
		length := len(buffer)
		var msg []byte
		if length <= bufferCapacity {
			msg = append(buffers.Get().([]byte), buffer...)
		} else {
			msg = append([]byte(nil), buffer...)
		}
		w.sent <- msg
		return length, nil
	}
	// sync mode
	w.lock.Lock()
	defer w.lock.Unlock()
	return os.Stdout.Write(buffer)
}

func (w *asyncStdoutWriter) pollWriter() {
	defer close(w.closed)
	for buffer := range w.sent {
		_, _ = os.Stdout.Write(buffer)
		if cap(buffer) == bufferCapacity {
			buffers.Put(buffer[:0])
		}
	}
}

func (w *asyncStdoutWriter) syncMode(ctx context.Context) {
	// lock first
	w.lock.Lock()
	defer w.lock.Unlock()
	// switch to sync mode
	w.mode.Store(true)
	// close the async channel
	close(w.sent)
	// wait for writer routine return
	select {
	case <-w.closed:
	case <-ctx.Done():
	}
}
