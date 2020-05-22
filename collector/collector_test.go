// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tsuru/prometheus-conntrack/workload"
	workloadTesting "github.com/tsuru/prometheus-conntrack/workload/testing"
	check "gopkg.in/check.v1"
)

type fakeConntrack struct {
	calls int
	conns [][]*Conn
}

func (f *fakeConntrack) conntrack() ([]*Conn, error) {
	f.calls = f.calls + 1
	return f.conns[f.calls-1], nil
}

func (*S) TestCollector(c *check.C) {
	conntrack := &fakeConntrack{
		conns: [][]*Conn{
			{
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.5", DestinationPort: "2376", State: "ESTABLISHED", Protocol: "tcp"},
			},
			{
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
			},
			{
				{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
			},
		},
	}

	collector := New(
		workloadTesting.New("containerd", "container", []*workload.Workload{
			{Name: "my-container1", IP: "10.10.1.2", Labels: map[string]string{"label1": "val1"}},
		}),
		conntrack.conntrack,
	)
	prometheus.MustRegister(collector)
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/metrics", nil)
	c.Assert(err, check.IsNil)
	promhttp.Handler().ServeHTTP(rr, req)
	c.Assert(rr.Code, check.Equals, http.StatusOK)
	lines := strings.Split(rr.Body.String(), "\n")
	c.Assert(lines[2], check.Equals, `container_connections{container="my-container1",destination="192.168.50.4:2375",label_label1="val1",protocol="tcp",state="ESTABLISHED"} 2`)
	c.Assert(lines[3], check.Equals, `container_connections{container="my-container1",destination="192.168.50.5:2376",label_label1="val1",protocol="tcp",state="ESTABLISHED"} 1`)

	req, err = http.NewRequest("GET", "/metrics", nil)
	c.Assert(err, check.IsNil)
	rr = httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rr, req)
	c.Assert(rr.Code, check.Equals, http.StatusOK)
	lines = strings.Split(rr.Body.String(), "\n")
	c.Assert(lines[2], check.Equals, `container_connections{container="my-container1",destination="192.168.50.4:2375",label_label1="val1",protocol="tcp",state="ESTABLISHED"} 1`)
	c.Assert(lines[3], check.Equals, `container_connections{container="my-container1",destination="192.168.50.5:2376",label_label1="val1",protocol="tcp",state="ESTABLISHED"} 0`)

	req, err = http.NewRequest("GET", "/metrics", nil)
	c.Assert(err, check.IsNil)
	rr = httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rr, req)
	c.Assert(rr.Code, check.Equals, http.StatusOK)
	lines = strings.Split(rr.Body.String(), "\n")
	c.Assert(lines[2], check.Equals, `container_connections{container="my-container1",destination="192.168.50.4:2375",label_label1="val1",protocol="tcp",state="ESTABLISHED"} 1`)
	c.Assert(lines[3], check.Not(check.Equals), `container_connections{container="my-container1",destination="192.168.50.5:2376",label_label1="val1",protocol="tcp",state="ESTABLISHED"} 1`)
}

func (s *S) BenchmarkCollector(c *check.C) {
	conns := []*Conn{
		{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.3", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2374", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.6", DestinationPort: "2376", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.3", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.7", DestinationPort: "2376", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.3", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2374", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.6", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.5", DestinationPort: "2376", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.3", SourcePort: "33404", DestinationIP: "192.168.50.5", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.2", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2374", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.10.1.1", SourcePort: "33404", DestinationIP: "192.168.50.6", DestinationPort: "2376", State: "ESTABLISHED", Protocol: "tcp"},
	}

	conntrack := func() ([]*Conn, error) {
		return conns, nil
	}
	collector := New(
		workloadTesting.New("containerd", "container", []*workload.Workload{
			{Name: "my-container1", IP: "10.10.1.2"},
			{Name: "my-container2", IP: "10.10.1.3"},
		}),
		conntrack,
	)
	ch := make(chan prometheus.Metric)
	go func() {
		for _ = range ch {
		}
	}()
	for n := 0; n < c.N; n++ {
		collector.Collect(ch)
	}
	close(ch)
}
