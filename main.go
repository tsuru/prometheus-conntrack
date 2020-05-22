// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"

	_ "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tsuru/prometheus-conntrack/collector"
)

func main() {
	addr := flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	endpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "Docker endpoint.")
	protocol := flag.String("protocol", "", "Protocol to track connections. Defaults to all.")
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Fetching containers from %s...\n", *endpoint)
	collector := collector.New(
		collector.NewDockerContainerLister(*endpoint),
		collector.NewConntrack(*protocol),
	)
	prometheus.MustRegister(collector)
	log.Printf("HTTP server listening at %s...\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
