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
	"github.com/tsuru/prometheus-conntrack/workload"
	"github.com/tsuru/prometheus-conntrack/workload/docker"
	"github.com/tsuru/prometheus-conntrack/workload/kubelet"
)

func main() {
	addr := flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	dockerEndpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "Docker endpoint.")
	kubeletEndpoint := flag.String("kubelet-endpoint", "https://127.0.0.1:10250/pods", "Kubelet endpoint.")
	protocol := flag.String("protocol", "", "Protocol to track connections. Defaults to all.")
	engineName := flag.String("engine", "docker", "Engine to track local workload addresses. Defaults to docker.")
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())

	var engine workload.Engine
	if *engineName == "kubelet" {
		log.Printf("Fetching workload from kubelet: %s...\n", *kubeletEndpoint)
		engine = kubelet.NewEngine(*kubeletEndpoint)
	} else {
		log.Printf("Fetching workload from docker: %s...\n", *dockerEndpoint)
		engine = docker.NewEngine(*dockerEndpoint)
	}

	conntrack := collector.NewConntrack(*protocol)
	collector := collector.New(engine, conntrack)
	prometheus.MustRegister(collector)
	log.Printf("HTTP server listening at %s...\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
