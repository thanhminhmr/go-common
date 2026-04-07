/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package tls

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/thanhminhmr/go-common/tcp"
)

type ServerHandler interface {
	Handle(ctx context.Context, conn *tls.Conn) error
}

type ServerHandlerFunc func(ctx context.Context, conn *tls.Conn) error

func (f ServerHandlerFunc) Handle(ctx context.Context, conn *tls.Conn) error {
	return f(ctx, conn)
}

func NewServerHandler(config *tls.Config, handler ServerHandler) tcp.ServerHandlerFunc {
	return func(ctx context.Context, conn *net.TCPConn) error {
		return handler.Handle(ctx, tls.Server(conn, config))
	}
}
