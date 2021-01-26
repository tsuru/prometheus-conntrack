// Copyright 2020 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kubelet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWorkloads(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(&podList{
			Items: []pod{
				{
					Metadata: podMetadata{
						Name:      "my-pod",
						Namespace: "tsuru",
						Labels: map[string]string{
							"version": "v3",
						},
					},
					Status: podStatus{
						PodIP: "10.27.24.12",
					},
				},
				{
					Metadata: podMetadata{
						Name:      "my-host-pod",
						Namespace: "kube",
						Labels: map[string]string{
							"version": "v3",
						},
					},
					Spec: podSpec{
						HostNetwork: true,
					},
					Status: podStatus{
						PodIP: "171.1.1.2",
					},
				},
			},
		})
		require.NoError(t, err)
	}))
	defer ts.Close()

	engine, err := NewEngine(Opts{Endpoint: ts.URL})
	require.NoError(t, err)
	workloads, err := engine.Workloads()
	require.NoError(t, err)
	assert.Len(t, workloads, 1)
	assert.Equal(t, workloads[0].Name, "my-pod")
	assert.Equal(t, workloads[0].IP, "10.27.24.12")
	assert.Equal(t, workloads[0].Labels, map[string]string{
		"pod_namespace": "tsuru",
		"version":       "v3",
	})
}
