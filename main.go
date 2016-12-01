// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	addr := flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	endpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "The address to the docker api.")
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())
	conns := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "container_outbound_connections",
		Help: "Number of outbound connections by container and destination",
	},
		[]string{
			"id",
			"container_label_tsuru_app_name",
			"container_label_tsuru_process_name",
			"dst",
		},
	)
	prometheus.MustRegister(conns)
	go run(*endpoint, conns)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func run(endpoint string, connsGauge *prometheus.GaugeVec) {
	for {
		containers, err := listContainers(endpoint)
		if err != nil {
			log.Print(err)
		}
		conns, err := conntrack()
		if err != nil {
			log.Print(err)
		}
		for _, c := range containers {
			if c.State != "" && c.State != "running" {
				continue
			}
			count := make(map[string]int)
			cont, err := inspect(c.ID, endpoint)
			if err != nil {
				log.Print(err)
			}
			for _, conn := range conns {
				value := ""
				switch cont.NetworkSettings.IPAddress {
				case conn.SourceIP:
					value = conn.DestinationIP + ":" + conn.DestinationPort
				case conn.DestinationIP:
					value = conn.SourceIP + ":" + conn.SourcePort
				}
				if value != "" {
					count[value] = count[value] + 1
				}
			}
			for k, v := range count {
				connsGauge.With(prometheus.Labels{
					"id": cont.ID,
					"container_label_tsuru_app_name":     cont.Config.Labels["container_label_tsuru_app_name"],
					"container_label_tsuru_process_name": cont.Config.Labels["container_label_tsuru_process_name"],
					"dst": k,
				}).Set(float64(v))
			}
		}
	}
}
