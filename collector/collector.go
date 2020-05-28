// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tsuru/prometheus-conntrack/workload"

	promstrutil "github.com/prometheus/prometheus/util/strutil"
)

var (
	additionalLabels    = []string{"state", "protocol", "destination", "destination_name", "direction"}
	unusedConnectionTTL = 2 * time.Minute
	desc                = prometheus.NewDesc("container_connections", "Number of outbound connections by destination and state", []string{"id", "name"}, nil)
)

type ConnDirection string

var (
	OutcommingConnection = ConnDirection("outcomming")
	IncommingConnection  = ConnDirection("incomming")
)

type Conntrack func() ([]*Conn, error)

type destination struct {
	ip   string
	port string
}

func (d *destination) String() string {
	return d.ip + ":" + d.port
}

type accumulatorKey struct {
	workload    string
	state       string
	protocol    string
	destination destination
	direction   ConnDirection
}

type ConntrackCollector struct {
	engine                  workload.Engine
	conntrack               Conntrack
	workloadLabels          []string
	sanitizedWorkloadLabels []string
	metricTupleSize         int
	fetchWorkloads          prometheus.Counter
	fetchWorkloadFailures   prometheus.Counter
	dnsCache                DNSCache
	// lastUsedTuples works such as a TTL, prometheus needs to know when connection is closed
	// then we will inform metric with 0 value for a while
	lastUsedTuples sync.Map
}

func New(engine workload.Engine, conntrack Conntrack, workloadLabels []string, dnsCache DNSCache) *ConntrackCollector {
	sanitizedWorkloadLabels := []string{engine.Kind()}
	for _, workloadLabel := range workloadLabels {
		sanitizedWorkloadLabels = append(sanitizedWorkloadLabels, "label_"+promstrutil.SanitizeLabelName(workloadLabel))
	}
	for _, label := range additionalLabels {
		sanitizedWorkloadLabels = append(sanitizedWorkloadLabels, label)
	}

	if dnsCache == nil {
		dnsCache = newDNSCache()
	}

	collector := &ConntrackCollector{
		engine:                  engine,
		conntrack:               conntrack,
		workloadLabels:          workloadLabels,
		sanitizedWorkloadLabels: sanitizedWorkloadLabels,
		metricTupleSize:         1 + len(workloadLabels) + len(additionalLabels),
		dnsCache:                dnsCache,
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

	go collector.metricCleaner()
	return collector
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
	counts := map[accumulatorKey]int{}
	workloadMap := map[string]*workload.Workload{}
	for _, workload := range workloads {
		for _, conn := range conns {
			var d destination
			var direction ConnDirection
			switch workload.IP {
			case conn.OriginIP:
				d = destination{conn.ReplyIP, conn.ReplyPort}
				direction = OutcommingConnection
			case conn.ReplyIP:
				d = destination{"", conn.ReplyPort}
				direction = IncommingConnection
			default:
				continue
			}

			key := accumulatorKey{
				workload:    workload.Name,
				protocol:    conn.Protocol,
				state:       conn.State,
				destination: d,
				direction:   direction,
			}
			counts[key] = counts[key] + 1
		}

		workloadMap[workload.Name] = workload
	}

	now := time.Now().UTC()
	for accumulatorKey := range counts {
		c.lastUsedTuples.Store(accumulatorKey, now)
	}

	c.sendMetrics(counts, workloadMap, ch)
}

func (c *ConntrackCollector) metricCleaner() {
	for {
		c.performMetricCleaner()
		time.Sleep(unusedConnectionTTL)
	}
}

func (c *ConntrackCollector) performMetricCleaner() {
	now := time.Now().UTC()
	c.lastUsedTuples.Range(func(key, lastUsed interface{}) bool {
		accumulator := key.(accumulatorKey)
		lastUsedTime := lastUsed.(time.Time)

		if now.After(lastUsedTime.Add(unusedConnectionTTL)) {
			c.lastUsedTuples.Delete(accumulator)
		}
		return true
	})
}

func (c *ConntrackCollector) sendMetrics(counts map[accumulatorKey]int, workloads map[string]*workload.Workload, ch chan<- prometheus.Metric) {
	c.lastUsedTuples.Range(func(key, _ interface{}) bool {
		accumulator := key.(accumulatorKey)
		count := counts[accumulator]
		workload := workloads[accumulator.workload]
		values := make([]string, c.metricTupleSize)

		values[0] = workload.Name
		i := 1
		for _, k := range c.workloadLabels {
			values[i] = workload.Labels[k]
			i++
		}
		desc := prometheus.NewDesc("conntrack_workload_connections", "Number of outbound connections by destination and state", c.sanitizedWorkloadLabels, nil)
		values[i] = accumulator.state
		values[i+1] = accumulator.protocol
		values[i+2] = accumulator.destination.String()
		if accumulator.destination.ip != "" {
			values[i+3] = c.dnsCache.ResolveIP(accumulator.destination.ip)
		}
		values[i+4] = string(accumulator.direction)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(count), values...)
		return true
	})
}
