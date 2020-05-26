// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

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
	protocol := flag.String("protocol", "", "Protocol to track connections. Defaults to all.")
	engineName := flag.String("engine", "docker", "Engine to track local workload addresses. Defaults to docker.")
	workloadLabelsString := flag.String("workload-labels", "", "Labels to extract from workload. ie (tsuru.io/app-name,tsuru.io/process-name)")

	dockerEndpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "Docker endpoint.")
	kubeletEndpoint := flag.String("kubelet-endpoint", "https://127.0.0.1:10250/pods", "Kubelet endpoint.")
	kubeletKey := flag.String("kubelet-key", "", "Path to a key to authenticate on kubelet.")
	kubeletCert := flag.String("kubelet-cert", "", "Path to a certificate to authenticate on kubelet.")
	kubeletCA := flag.String("kubelet-ca", "", "Path to a CA to authenticate on kubelet.")

	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())

	var engine workload.Engine
	var err error
	if *engineName == "kubelet" {
		log.Printf("Fetching workload from kubelet: %s...\n", *kubeletEndpoint)
		engine, err = kubelet.NewEngine(kubelet.Opts{
			Endpoint: *kubeletEndpoint,
			Key:      *kubeletKey,
			Cert:     *kubeletCert,
			CA:       *kubeletCA,
		})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("Fetching workload from docker: %s...\n", *dockerEndpoint)
		engine = docker.NewEngine(*dockerEndpoint)
	}

	workloadLabels := strings.Split(*workloadLabelsString, ",")
	conntrack := collector.NewConntrack(*protocol)
	collector := collector.New(engine, conntrack, workloadLabels)
	prometheus.MustRegister(collector)
	log.Printf("HTTP server listening at %s...\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
