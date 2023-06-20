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

	sysctl "github.com/lorenzosaino/go-sysctl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tsuru/prometheus-conntrack/collector"
	"github.com/tsuru/prometheus-conntrack/workload"
	"github.com/tsuru/prometheus-conntrack/workload/docker"
	"github.com/tsuru/prometheus-conntrack/workload/kubelet"
)

const conntrackTimestampFlag = "net.netfilter.nf_conntrack_timestamp"

func main() {
	addr := flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	protocol := flag.String("protocol", "", "Protocol to track connections. Defaults to all.")
	engineName := flag.String("engine", "docker", "Engine to track local workload addresses. Defaults to docker.")
	workloadLabelsString := flag.String("workload-labels", "", "Labels to extract from workload. ie (tsuru.io/app-name,tsuru.io/process-name)")
	cidrClassesString := flag.String("cidr-classes", "", "CIDRs to extract labels. ie (10.0.0.0/8=internal,0.0.0.0/0=internet)")

	trackSynSent := flag.Bool("track-syn-sent", false, "Turn on track of stuck connections with syn-sent, will enable automatically the net.netfilter.nf_conntrack_timestamp flag on kernel.")

	dockerEndpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "Docker endpoint.")
	kubeletEndpoint := flag.String("kubelet-endpoint", "https://127.0.0.1:10250/pods", "Kubelet endpoint.")
	kubeletKey := flag.String("kubelet-key", "", "Path to a key to authenticate on kubelet.")
	kubeletCert := flag.String("kubelet-cert", "", "Path to a certificate to authenticate on kubelet.")
	kubeletCA := flag.String("kubelet-ca", "", "Path to a CA to authenticate on kubelet.")
	kubeletToken := flag.String("kubelet-token", "", "Path the token to authenticate on kubelet.")
	insecureSkipTLSVerify := flag.Bool("insecure-skip-tls-verify", false, "controls whether a client verifies the server's certificate chain and host name.")

	flag.Parse()

	if *trackSynSent {
		enableConntrackTimestamps()
	}

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
			Token:    *kubeletToken,

			InsecureSkipVerify: *insecureSkipTLSVerify,
		})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("Fetching workload from docker: %s...\n", *dockerEndpoint)
		engine = docker.NewEngine(*dockerEndpoint)
	}

	workloadLabels := strings.Split(*workloadLabelsString, ",")

	cidrClasses := map[string]string{}
	if *cidrClassesString != "" {
		for _, keyPairStr := range strings.Split(*cidrClassesString, ",") {
			keyPair := strings.SplitN(keyPairStr, "=", 2)

			if len(keyPair) != 2 {
				log.Fatalf("Invalid cidr key pair: %s", keyPairStr)
			}

			cidrClasses[keyPair[0]] = keyPair[1]
		}
	}

	classifier, err := collector.NewCIDRClassifier(cidrClasses)
	if err != nil {
		log.Fatal(err)
	}

	conntrack := collector.NewConntrack(*protocol)
	collector, err := collector.New(engine, conntrack, workloadLabels, nil, classifier)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(collector)
	log.Printf("HTTP server listening at %s...\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func enableConntrackTimestamps() {
	val, err := sysctl.Get(conntrackTimestampFlag)
	if err != nil {
		log.Printf("Could not get status of %s, err: %s", conntrackTimestampFlag, err.Error())
		return
	}
	if val == "1" {
		log.Printf("Flag %s is already turned on", conntrackTimestampFlag)
		return
	}
	err = sysctl.Set(conntrackTimestampFlag, "1")
	if err != nil {
		log.Printf("Could not set status of %s, err: %s", conntrackTimestampFlag, err.Error())
		return
	}

	log.Printf("Flag %s was turned on", conntrackTimestampFlag)

}
