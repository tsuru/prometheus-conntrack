// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsuru/prometheus-conntrack/workload"
	workloadTesting "github.com/tsuru/prometheus-conntrack/workload/testing"
)

type fakeDNSCache struct{}

var fakeResolvConf = map[string]string{
	"127.0.0.1":    "localhost",
	"192.168.50.4": "alice-service",
	"192.168.50.5": "bob-service",
	"10.10.1.2":    "john-service",
	"10.100.1.2":   "gateway-service",
}

func (f *fakeDNSCache) ResolveIP(ip string) string {
	return fakeResolvConf[ip]
}

type fakeConntrack struct {
	calls int
	conns [][]*Conn
}

func (f *fakeConntrack) conntrack() ([]*Conn, error) {
	f.calls = f.calls + 1
	return f.conns[f.calls-1], nil
}

func TestCollector(t *testing.T) {
	conntrack := &fakeConntrack{
		conns: [][]*Conn{
			{
				{OriginIP: "10.100.1.2", OriginPort: 33404, DestIP: "192.165.50.4", DestPort: 443, State: "ESTABLISHED", Protocol: "tcp"},
				{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
				{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
				{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.5", DestPort: 2376, State: "ESTABLISHED", Protocol: "tcp"},
				{OriginIP: "192.168.50.5", OriginPort: 33404, DestIP: "10.10.1.2", DestPort: 7070, State: "ESTABLISHED", Protocol: "tcp"},
			},
			{
				{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
			},
		},
	}

	collector := New(
		workloadTesting.New("containerd", "container", []*workload.Workload{
			{Name: "my-container1", IP: "10.10.1.2", Labels: map[string]string{"label1": "val1", "app": "app1"}},
		}),
		conntrack.conntrack,
		[]string{"app"},
		&fakeDNSCache{},
	)
	prometheus.MustRegister(collector)
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	promhttp.Handler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	lines := strings.Split(rr.Body.String(), "\n")
	assert.Contains(t, lines, `conntrack_workload_connections{container="my-container1",destination="192.168.50.4:2375",destination_name="alice-service",direction="outgoing",label_app="app1",protocol="tcp",state="ESTABLISHED"} 2`)
	assert.Contains(t, lines, `conntrack_workload_connections{container="my-container1",destination="192.168.50.5:2376",destination_name="bob-service",direction="outgoing",label_app="app1",protocol="tcp",state="ESTABLISHED"} 1`)
	assert.Contains(t, lines, `conntrack_workload_connections{container="my-container1",destination=":7070",destination_name="",direction="incoming",label_app="app1",protocol="tcp",state="ESTABLISHED"} 1`)

	req, err = http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	lines = strings.Split(rr.Body.String(), "\n")
	assert.Contains(t, lines, `conntrack_workload_connections{container="my-container1",destination="192.168.50.4:2375",destination_name="alice-service",direction="outgoing",label_app="app1",protocol="tcp",state="ESTABLISHED"} 1`)
	assert.Contains(t, lines, `conntrack_workload_connections{container="my-container1",destination="192.168.50.5:2376",destination_name="bob-service",direction="outgoing",label_app="app1",protocol="tcp",state="ESTABLISHED"} 0`)
}

func TestPerformMetricClean(t *testing.T) {
	collector := &ConntrackCollector{}
	now := time.Now().UTC()
	collector.lastUsedWorkloadTuples.Store(accumulatorKey{workload: "w1", state: "estab", protocol: "tcp", destination: destination{ip: "blah"}}, now.Add(time.Minute*-60))
	collector.lastUsedWorkloadTuples.Store(accumulatorKey{workload: "w2", state: "estab", protocol: "tcp", destination: destination{ip: "blah"}}, now)
	collector.lastUsedWorkloadTuples.Store(accumulatorKey{workload: "w3", state: "estab", protocol: "tcp", destination: destination{ip: "blah"}}, now.Add(time.Minute*60))

	collector.performMetricCleaner()

	keys := []string{}
	collector.lastUsedWorkloadTuples.Range(func(key, lastUsed interface{}) bool {
		keys = append(keys, key.(accumulatorKey).workload)
		return true
	})
	sort.Strings(keys)
	assert.Equal(t, []string{"w2", "w3"}, keys)
}

func BenchmarkCollector(b *testing.B) {
	conns := []*Conn{
		{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.3", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2374, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.6", DestPort: 2376, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.3", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.7", DestPort: 2376, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.3", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2374, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.6", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.5", DestPort: 2376, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.3", OriginPort: 33404, DestIP: "192.168.50.5", DestPort: 2375, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.2", OriginPort: 33404, DestIP: "192.168.50.4", DestPort: 2374, State: "ESTABLISHED", Protocol: "tcp"},
		{OriginIP: "10.10.1.1", OriginPort: 33404, DestIP: "192.168.50.6", DestPort: 2376, State: "ESTABLISHED", Protocol: "tcp"},
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
		[]string{},
		&fakeDNSCache{},
	)
	ch := make(chan prometheus.Metric)
	for n := 0; n < b.N; n++ {
		collector.Collect(ch)
	}
	close(ch)
}
