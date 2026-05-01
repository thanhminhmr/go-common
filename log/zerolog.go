/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package log

import (
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Config struct {
	TimestampFormat     string `env:"LOGGER_TIMESTAMP_FORMAT" validator:"required" default:"2006-01-02T15:04:05.000000000Z07:00"`
	TimestampResolution string `env:"LOGGER_TIMESTAMP_RESOLUTION" validator:"oneof=seconds milliseconds microseconds nanoseconds" default:"nanoseconds"`
	EnableSyncMode      bool   `env:"LOGGER_ENABLE_SYNC_MODE"`
}

func ConsoleLogger(lifecycle fx.Lifecycle, config *Config) *zerolog.Logger {
	// config global resolution
	switch config.TimestampResolution {
	case "seconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	case "milliseconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	case "microseconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	case "nanoseconds":
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	}
	// create the writer
	writer := io.Writer(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: config.TimestampFormat,
	})
	// check if sync mode is enabled
	if config.EnableSyncMode {
		// wrap with sync writer
		writer = zerolog.SyncWriter(writer)
	} else {
		// wrap with async writer
		wrapped := asyncWriter{
			writer: writer,
			sent:   make(chan []byte, 1024),
			closed: make(chan struct{}),
		}
		// enable poll and switch to sync mode on shutdown
		go wrapped.pollWriter()
		lifecycle.Append(fx.Hook{OnStop: wrapped.syncMode})
		writer = &wrapped
	}
	// create the logger
	logger := zerolog.New(writer).With().Timestamp().Caller().Logger()
	// set the logger as default logger
	zerolog.DefaultContextLogger = &logger
	return &logger
}

var buffers = sync.Pool{New: func() any { return make([]byte, 0, 1024) }}

type asyncWriter struct {
	writer io.Writer
	sent   chan []byte
	closed chan struct{}
	lock   sync.RWMutex
	mode   atomic.Bool
}

func (w *asyncWriter) Write(buffer []byte) (n int, err error) {
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
		if length <= 1024 {
			msg = buffers.Get().([]byte)[:length]
			copy(msg, buffer)
		} else {
			msg = append([]byte(nil), buffer...)
		}
		w.sent <- msg
		return length, nil
	}
	// sync mode
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.writer.Write(buffer)
}

func (w *asyncWriter) pollWriter() {
	defer close(w.closed)
	for buffer := range w.sent {
		_, _ = w.writer.Write(buffer)
		if cap(buffer) == 1024 {
			buffers.Put(buffer)
		}
	}
}

func (w *asyncWriter) syncMode(ctx context.Context) error {
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
	return ctx.Err()
}
