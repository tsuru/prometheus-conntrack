// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"log"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tsuru/prometheus-conntrack/workload"
)

var (
	additionalLabels = []string{"state", "protocol", "destination"}
	desc             = prometheus.NewDesc("container_connections", "Number of outbound connections by destionation and state", []string{"id", "name"}, nil)
)

type Conntrack func() ([]*Conn, error)

type ConntrackCollector struct {
	engine    workload.Engine
	conntrack Conntrack
	sync.Mutex
	connCount             map[string]map[string]int
	workloads             map[string]*workload.Workload
	workloadLabels        []string
	fetchWorkloads        prometheus.Counter
	fetchWorkloadFailures prometheus.Counter
}

func New(engine workload.Engine, conntrack Conntrack, workloadLabels []string) *ConntrackCollector {
	return &ConntrackCollector{
		engine:         engine,
		conntrack:      conntrack,
		connCount:      make(map[string]map[string]int),
		workloads:      make(map[string]*workload.Workload),
		workloadLabels: workloadLabels,
		fetchWorkloads: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "conntrack",
			Subsystem: "workload",
			Name:      "fetch_total",
			Help:      "Number of fetchs to discover workloads",
		}),
		fetchWorkloadFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "conntrack",
			Subsystem: "workload",
			Name:      "failures_total",
			Help:      "Number of failures to get workloads",
		}),
	}
}

func (c *ConntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- desc
}

func (c *ConntrackCollector) Collect(ch chan<- prometheus.Metric) {
	c.fetchWorkloads.Inc()
	ch <- c.fetchWorkloads
	workloads, err := c.engine.Workloads()
	if err != nil {
		c.fetchWorkloadFailures.Inc()
		ch <- c.fetchWorkloadFailures
		log.Print(err)
		return
	}
	ch <- c.fetchWorkloadFailures

	conns, err := c.conntrack()
	if err != nil {
		log.Print(err)
		return
	}
	counts, currWorkloads := c.getState()
	for _, workload := range workloads {
		for _, conn := range conns {
			value := ""
			switch workload.IP {
			case conn.SourceIP:
				value = conn.DestinationIP + ":" + conn.DestinationPort
			case conn.DestinationIP:
				value = conn.SourceIP + ":" + conn.SourcePort
			}
			if value != "" {
				key := conn.State + "-" + conn.Protocol + "-" + value
				if counts[workload.Name] == nil {
					counts[workload.Name] = make(map[string]int)
				}
				counts[workload.Name][key] = counts[workload.Name][key] + 1
			}
			currWorkloads[workload.Name] = workload
		}
	}
	c.setState(counts, currWorkloads)
	c.sendMetrics(counts, currWorkloads, ch)
}

func (c *ConntrackCollector) getState() (map[string]map[string]int, map[string]*workload.Workload) {
	c.Lock()
	defer c.Unlock()
	copyWorkloads := make(map[string]*workload.Workload)
	for _, workload := range c.workloads {
		copyWorkloads[workload.Name] = c.workloads[workload.Name]
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
			delete(copyWorkloads, k)
			continue
		}
		copy[k] = innerCopy
	}
	return copy, copyWorkloads
}

func (c *ConntrackCollector) setState(count map[string]map[string]int, workloads map[string]*workload.Workload) {
	c.Lock()
	defer c.Unlock()
	c.connCount = count
	for k, v := range workloads {
		c.workloads[k] = v
	}
}

func (c *ConntrackCollector) sendMetrics(metrics map[string]map[string]int, workloads map[string]*workload.Workload, ch chan<- prometheus.Metric) {
	for worloadID, count := range metrics {
		workload := workloads[worloadID]
		labels := make([]string, 1+len(c.workloadLabels)+len(additionalLabels))
		values := make([]string, 1+len(c.workloadLabels)+len(additionalLabels))

		labels[0] = c.engine.Kind()
		values[0] = workload.Name
		i := 1
		for _, k := range c.workloadLabels {
			labels[i] = "label_" + sanitizeLabelName(k)
			values[i] = workload.Labels[k]
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

func sanitizeLabelName(name string) string {
	return strings.Replace(name, ".", "_", -1)
}
