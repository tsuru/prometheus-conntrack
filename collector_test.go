// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	check "gopkg.in/check.v1"
)

func (*S) TestCollector(c *check.C) {
	collector := &ConntrackCollector{
		containerLister: func() ([]*docker.Container, error) {
			return []*docker.Container{{
				ID:              "id",
				Name:            "name",
				Config:          &docker.Config{Image: "image", Labels: map[string]string{"label1": "val1"}},
				NetworkSettings: &docker.NetworkSettings{IPAddress: "10.10.1.2"},
			}}, nil
		},
		conntrack: func() ([]conn, error) {
			return []conn{
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.5", DestinationPort: "2376", State: "ESTABLISHED", Protocol: "tcp"},
			}, nil
		},
	}
	prometheus.MustRegister(collector)
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/metrics", nil)
	c.Assert(err, check.IsNil)
	promhttp.Handler().ServeHTTP(rr, req)
	c.Assert(rr.Code, check.Equals, http.StatusOK)
	lines := strings.Split(rr.Body.String(), "\n")
	c.Assert(lines[2], check.Equals, `container_connections{container_label_label1="val1",destination="192.168.50.4:2375",id="id",image="image",name="name",protocol="tcp",state="ESTABLISHED"} 2`)
	c.Assert(lines[3], check.Equals, `container_connections{container_label_label1="val1",destination="192.168.50.5:2376",id="id",image="image",name="name",protocol="tcp",state="ESTABLISHED"} 1`)
}
