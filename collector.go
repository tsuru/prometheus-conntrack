// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"regexp"
	"strings"
	"sync"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/prometheus/client_golang/prometheus"
)

type ConntrackCollector struct {
	containerLister func() ([]*docker.Container, error)
	conntrack       func() ([]conn, error)
	sync.Mutex
	connCount  map[string]map[string]int
	containers map[string]*docker.Container
}

func (c *ConntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("container_connections", "Number of outbound connections by destionation and state", []string{"id", "name"}, nil)
}

func (c *ConntrackCollector) Collect(ch chan<- prometheus.Metric) {
	counts, currContainers := c.getState()
	containers, err := c.containerLister()
	if err != nil {
		log.Print(err)
	}
	conns, err := c.conntrack()
	if err != nil {
		log.Print(err)
	}
	for _, container := range containers {
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
				if counts[container.ID] == nil {
					counts[container.ID] = make(map[string]int)
				}
				counts[container.ID][key] = counts[container.ID][key] + 1
			}
			currContainers[container.ID] = *container
		}
	}
	c.setState(counts, currContainers)
	c.sendMetrics(counts, ch)
}

func (c *ConntrackCollector) getState() (map[string]map[string]int, map[string]docker.Container) {
	c.Lock()
	defer c.Unlock()
	copyCont := make(map[string]docker.Container)
	for _, cont := range c.containers {
		copyCont[cont.ID] = *cont
	}
	copy := make(map[string]map[string]int)
	for k, v := range c.connCount {
		innerCopy := make(map[string]int)
		for ik, iv := range v {
			if iv == 0 {
				continue
			}
			innerCopy[ik] = 0
		}
		if len(innerCopy) == 0 {
			delete(copyCont, k)
			continue
		}
		copy[k] = innerCopy
	}
	return copy, copyCont
}

func (c *ConntrackCollector) setState(count map[string]map[string]int, containers map[string]docker.Container) {
	c.Lock()
	defer c.Unlock()
	c.connCount = count
	for k, v := range containers {
		c.containers[k] = &v
	}
}

func (c *ConntrackCollector) sendMetrics(metrics map[string]map[string]int, ch chan<- prometheus.Metric) {
	for contID, count := range metrics {
		labels, values := []string{}, []string{}
		for k, v := range containerLabels(c.containers[contID]) {
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
