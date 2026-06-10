/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package common

import "unsafe"

// Check on compile time that [NoCopy] is indeed zero size.
var _ = [1]int{unsafe.Sizeof(NoCopy{}): 0}

// NoCopy is a zero-sized type that tells the compiler that it should not be
// copied (by implementing [sync.Locker]).
type NoCopy struct{}

func (n *NoCopy) Lock()   {}
func (n *NoCopy) Unlock() {}
