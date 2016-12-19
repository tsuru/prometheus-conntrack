// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"regexp"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/prometheus/client_golang/prometheus"
)

var additionalLabels = []string{"state", "protocol", "destination"}

type ConntrackCollector struct {
	containerLister func() ([]*docker.Container, error)
	conntrack       func() ([]conn, error)
}

func (c *ConntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("container_connections", "Number of outbound connections by destionation and state", []string{"id", "name"}, nil)
}

func (c *ConntrackCollector) Collect(ch chan<- prometheus.Metric) {
	containers, err := c.containerLister()
	if err != nil {
		log.Print(err)
	}
	conns, err := c.conntrack()
	if err != nil {
		log.Print(err)
	}
	for _, container := range containers {
		count := make(map[string]int)
		for _, conn := range conns {
			value := ""
			switch container.NetworkSettings.IPAddress {
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
		labelsMap := containerLabels(container)
		labels := make([]string, len(labelsMap)+len(additionalLabels))
		values := make([]string, len(labelsMap)+len(additionalLabels))
		i := 0
		for k, v := range labelsMap {
			labels[i] = k
			values[i] = v
			i++
		}
		for _, l := range additionalLabels {
			labels[i] = l
			i++
		}
		i = i - len(additionalLabels)
		desc := prometheus.NewDesc("container_connections", "Number of outbound connections by destionation and state", labels, nil)
		for k, v := range count {
			keys := strings.SplitN(k, "-", 3)
			values[i] = keys[0]
			values[i+1] = keys[1]
			values[i+2] = keys[2]
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(v), values...)
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
