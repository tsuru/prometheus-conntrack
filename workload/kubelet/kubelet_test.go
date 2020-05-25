// Copyright 2020 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kubelet

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&S{})

func Test(t *testing.T) {
	check.TestingT(t)
}

type S struct{}

func (s *S) TestListWorkloads(c *check.C) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(&podList{
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
			},
		})
	}))
	defer ts.Close()

	engine := NewEngine(ts.URL)
	workloads, err := engine.Workloads()
	c.Assert(err, check.IsNil)
	c.Assert(len(workloads), check.Equals, 1)
	c.Assert(workloads[0].Name, check.Equals, "my-pod")
	c.Assert(workloads[0].IP, check.Equals, "10.27.24.12")
	c.Assert(workloads[0].Labels, check.DeepEquals, map[string]string{
		"pod_namespace": "tsuru",
		"version":       "v3",
	})
}
