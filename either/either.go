/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package either

import "unsafe"

type Either[Left, Right any] struct {
	state int
	ptr   unsafe.Pointer
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

func (b Either[Left, Right]) Neither() bool {
	return b.state == 0
}

func Left[Left, Right any](left Left) Either[Left, Right] {
	return Either[Left, Right]{
		state: -1,
		ptr:   unsafe.Pointer(&left),
	}
}

func Right[Left, Right any](right Right) Either[Left, Right] {
	return Either[Left, Right]{
		state: 1,
		ptr:   unsafe.Pointer(&right),
	}
}
