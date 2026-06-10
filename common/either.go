/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common

import "unsafe"

type Either[Left, Right any] struct {
	ptr   unsafe.Pointer
	state int8
}

func (b Either[Left, Right]) Left() (left Left, exists bool) {
	if b.state < 0 {
		return *(*Left)(b.ptr), true
	}
	return left, false
}

func (b Either[Left, Right]) Right() (right Right, exists bool) {
	if b.state > 0 {
		return *(*Right)(b.ptr), true
	}
	return right, false
}

func (b Either[Left, Right]) Either() (left Left, right Right, state int) {
	if b.state < 0 {
		return *(*Left)(b.ptr), right, -1
	} else if b.state > 0 {
		return left, *(*Right)(b.ptr), 1
	}
	return
}

func (b Either[Left, Right]) Neither() bool {
	return b.state == 0
}

func Left[Left, Right any](left Left) Either[Left, Right] {
	return Either[Left, Right]{
		ptr:   unsafe.Pointer(&left),
		state: -1,
	}
}

func Right[Left, Right any](right Right) Either[Left, Right] {
	return Either[Left, Right]{
		ptr:   unsafe.Pointer(&right),
		state: 1,
	}
}
