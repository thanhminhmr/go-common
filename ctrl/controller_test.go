/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package ctrl_test

import (
	"context"
	"testing"
	"time"
	_ "unsafe"

	"github.com/thanhminhmr/go-common/ctrl"
	"github.com/thanhminhmr/go-exception"
)

//go:linkname setup github.com/thanhminhmr/go-common/ctrl.setup
func setup()

const private = "private"

func TestInitializerPanic(t *testing.T) {
	setup()
	defer exception.Recover(func(ex exception.Exception) { t.Fail() })
	ctrl.Run(func() { panic(private) })
}

func TestStrayRegister(t *testing.T) {
	setup()
	defer exception.Recover(func(ex exception.Exception) {})
	ctrl.Register(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) { return nil, nil })
	t.Fail()
}

func TestRegisterPanic(t *testing.T) {
	setup()
	defer exception.Recover(func(ex exception.Exception) { t.Fail() })
	ctrl.Run(func() {
		ctrl.Register(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) { panic(private) })
		t.Fail()
	})
}

func TestRunnerNominal(t *testing.T) {
	setup()
	defer exception.Recover(func(ex exception.Exception) { t.Fail() })
	ctrl.Run(func() {
		defer exception.Recover(func(ex exception.Exception) { t.Fail() })
		ctrl.Register(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) {
			return func(ctx context.Context, shutdown context.CancelFunc) {
				panic(private)
			}, nil
		})
		ctrl.Register(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) {
			return func(ctx context.Context, shutdown context.CancelFunc) {
				timer := time.After(time.Minute)
				select {
				case <-timer:
				case <-ctx.Done():
				}
			}, nil
		})
	})
}

func TestNominal(t *testing.T) {
	setup()
	defer exception.Recover(func(ex exception.Exception) { t.Fail() })
	ctrl.Run(func() {
		defer exception.Recover(func(ex exception.Exception) { t.Fail() })
		ctrl.Register(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) { return nil, nil })
	})
}

func TestShutdown(t *testing.T) {
	setup()
	defer exception.Recover(func(ex exception.Exception) { t.Fail() })
	ctrl.Run(func() {
		defer exception.Recover(func(ex exception.Exception) { t.Fail() })
		ctrl.Register(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) {
			return func(ctx context.Context, shutdown context.CancelFunc) {
				timer := time.After(time.Minute)
				select {
				case <-timer:
				case <-ctx.Done():
				}
			}, nil
		})
		ctrl.Register(func(ctx context.Context) (ctrl.Runner, ctrl.Cleaner) {
			return func(ctx context.Context, shutdown context.CancelFunc) {
				shutdown()
			}, nil
		})
	})
}
