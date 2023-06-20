// Copyright 2023 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIDRClassifier(t *testing.T) {
	c, err := NewCIDRClassifier(map[string]string{
		"10.0.0.0/8":  "internal",
		"10.1.1.0/24": "my-vpc",
		"0.0.0.0/0":   "internet",
	})

	require.NoError(t, err)

	assert.Equal(t, c.Classify("1.1.1.3"), "internet")
	assert.Equal(t, c.Classify("10.200.100.100"), "internal")
	assert.Equal(t, c.Classify("10.1.1.1"), "my-vpc")

}
