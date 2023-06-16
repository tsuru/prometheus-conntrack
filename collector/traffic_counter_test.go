// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrafficCounterOnce(t *testing.T) {

	tc := newTrafficCounter()
	tc.Inc(connTrafficKey{IP: "10.1.1.1", Port: 8000, Direction: OutgoingConnection}, 10, 10, 10)

	items := tc.List()

	assert.Len(t, items, 1)
	assert.Equal(t, items[0].IP, "10.1.1.1")
	assert.Equal(t, items[0].Port, uint16(8000))
	assert.Equal(t, items[0].Direction, OutgoingConnection)
	assert.Equal(t, items[0].ReplyCounter, uint64(10))
	assert.Equal(t, items[0].OriginCounter, uint64(10))
}

func TestTrafficCounterTwice(t *testing.T) {

	tc := newTrafficCounter()
	tc.Inc(connTrafficKey{IP: "10.1.1.1", Port: 8000, Direction: OutgoingConnection}, 10, 10, 10)
	tc.Inc(connTrafficKey{IP: "10.1.1.1", Port: 8000, Direction: OutgoingConnection}, 11, 110, 110)
	tc.Inc(connTrafficKey{IP: "10.1.1.1", Port: 8000, Direction: OutgoingConnection}, 10, 100, 100)

	items := tc.List()

	assert.Len(t, items, 1)
	assert.Equal(t, items[0].IP, "10.1.1.1")
	assert.Equal(t, items[0].Port, uint16(8000))
	assert.Equal(t, items[0].Direction, OutgoingConnection)
	assert.Equal(t, int(items[0].ReplyCounter), 210)
	assert.Equal(t, int(items[0].OriginCounter), 210)
}
