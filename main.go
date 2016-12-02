// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
	"regexp"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	addr := flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	endpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "The address to the docker api.")
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())
	collector := &ConntrackCollector{
		dockerEndpoint: *endpoint,
		conntrack:      conntrack,
	}
	prometheus.MustRegister(collector)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

type ConntrackCollector struct {
	dockerEndpoint string
	conntrack      func() ([]conn, error)
}

func (c *ConntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("container_connections", "Number of outbound connections by destionation and state", []string{"id", "name"}, nil)
}

func (c *ConntrackCollector) Collect(ch chan<- prometheus.Metric) {
	containers, err := listContainers(c.dockerEndpoint)
	if err != nil {
		log.Print(err)
	}
	conns, err := c.conntrack()
	if err != nil {
		log.Print(err)
	}
	for _, container := range containers {
		if container.State != "" && container.State != "running" {
			continue
		}
		count := make(map[string]int)
		cont, err := inspect(container.ID, c.dockerEndpoint)
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
				key := conn.State + "-" + conn.Protocol + "-" + value
				count[key] = count[key] + 1
			}
		}
		labels, values := []string{}, []string{}
		for k, v := range containerLabels(cont) {
			labels = append(labels, sanitizeLabelName(k))
			values = append(values, v)
		}
		labels = append(labels, "state", "protocol", "destination")
		for k, v := range count {
			keys := strings.SplitN(k, "-", 3)
			finalValues := append(values, keys...)
			desc := prometheus.NewDesc("container_connections", "Number of outbound connections by destionation and state", labels, nil)
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(v), finalValues...)
		}
	}
}

func containerLabels(container *docker.Container) map[string]string {
	labels := map[string]string{
		"id":    container.ID,
		"name":  container.Name,
		"image": container.Config.Image,
	}
	for k, v := range container.Config.Labels {
		labels["container_label_"+k] = v
	}
	return labels
}

var invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func sanitizeLabelName(name string) string {
	return invalidLabelCharRE.ReplaceAllString(name, "_")
}
