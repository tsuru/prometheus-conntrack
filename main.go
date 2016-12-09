// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	addr := flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	endpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "Docker endpoint.")
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Fetching containers from %s...\n", *endpoint)
	collector := &ConntrackCollector{
		containerLister: func() ([]*docker.Container, error) {
			return listContainers(*endpoint)
		},
		conntrack:  conntrack,
		connCount:  make(map[string]map[string]int),
		containers: make(map[string]*docker.Container),
	}
	prometheus.MustRegister(collector)
	log.Printf("HTTP server listening at %s...\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
