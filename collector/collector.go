// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tsuru/prometheus-conntrack/workload"

	promstrutil "github.com/prometheus/prometheus/util/strutil"
)

var (
	connectionLabels  = []string{"state", "protocol", "destination", "destination_name", "direction"}
	originBytesLabels = []string{"destination"}

	unusedConnectionTTL = 2 * time.Minute
)

type ConnDirection string

var (
	OutgoingConnection = ConnDirection("outgoing")
	IncomingConnection = ConnDirection("incoming")
)

type Conntrack func() ([]*Conn, error)

type destination struct {
	ip   string
	port uint16
}

func (d *destination) String() string {
	return fmt.Sprintf("%s:%d", d.ip, d.port)
}

type accumulatorKey struct {
	workload    string
	state       string
	protocol    string
	destination destination
	direction   ConnDirection
}

type ConntrackCollector struct {
	engine                    workload.Engine
	conntrack                 Conntrack
	workloadLabels            []string
	sanitizedWorkloadLabels   []string
	connectionMetricTupleSize int
	fetchWorkloads            prometheus.Counter
	fetchWorkloadFailures     prometheus.Counter
	dnsCache                  DNSCache
	// lastUsedWorkloadTuples works such as a TTL, prometheus needs to know when connection is closed
	// then we will inform metric with 0 value for a while
	lastUsedWorkloadTuples sync.Map

	trafficCounter *trafficCounter
}

func New(engine workload.Engine, conntrack Conntrack, workloadLabels []string, dnsCache DNSCache) *ConntrackCollector {
	sanitizedWorkloadLabels := []string{engine.Kind()}
	for _, workloadLabel := range workloadLabels {
		sanitizedWorkloadLabels = append(sanitizedWorkloadLabels, "label_"+promstrutil.SanitizeLabelName(workloadLabel))
	}

	if dnsCache == nil {
		dnsCache = newDNSCache()
	}

	collector := &ConntrackCollector{
		engine:                    engine,
		conntrack:                 conntrack,
		workloadLabels:            workloadLabels,
		sanitizedWorkloadLabels:   sanitizedWorkloadLabels,
		connectionMetricTupleSize: 1 + len(workloadLabels) + len(connectionLabels),
		dnsCache:                  dnsCache,
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

		trafficCounter: newTrafficCounter(),
	}

	go collector.metricCleaner()
	return collector
}

func (c *ConntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.workloadConnectionsDesc()
	ch <- c.workloadOriginBytesTotalDesc()
	ch <- c.workloadReplyBytesTotalDesc()
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

	c.trafficCounter.Lock()

	for _, workload := range workloads {
		for _, conn := range conns {
			var d destination
			var direction ConnDirection
			switch workload.IP {
			case conn.OriginIP:
				d = destination{conn.DestIP, conn.DestPort}
				direction = OutgoingConnection
			case conn.DestIP:
				d = destination{"", conn.DestPort}
				direction = IncomingConnection
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

			c.trafficCounter.Inc(connTrafficKey{Workload: workload.Name, IP: d.ip, Port: d.port}, conn.ID, conn.OriginBytes, conn.ReplyBytes)
		}

		workloadMap[workload.Name] = workload
	}

	c.trafficCounter.Unlock()

	now := time.Now().UTC()
	for accumulatorKey := range counts {
		c.lastUsedWorkloadTuples.Store(accumulatorKey, now)
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
	c.lastUsedWorkloadTuples.Range(func(key, lastUsed interface{}) bool {
		accumulator := key.(accumulatorKey)
		lastUsedTime := lastUsed.(time.Time)

		if now.After(lastUsedTime.Add(unusedConnectionTTL)) {
			c.lastUsedWorkloadTuples.Delete(accumulator)
		}
		return true
	})
}

func (c *ConntrackCollector) workloadConnectionsDesc() *prometheus.Desc {
	labels := []string{}
	labels = append(labels, c.sanitizedWorkloadLabels...)
	labels = append(labels, connectionLabels...)

	return prometheus.NewDesc("conntrack_workload_connections", "Number of outbound connections by destination and state", labels, nil)
}

func (c *ConntrackCollector) workloadOriginBytesTotalDesc() *prometheus.Desc {
	labels := []string{}
	labels = append(labels, c.sanitizedWorkloadLabels...)
	labels = append(labels, originBytesLabels...)

	return prometheus.NewDesc("conntrack_workload_origin_bytes_total", "Number of origin bytes", labels, nil)
}

func (c *ConntrackCollector) workloadReplyBytesTotalDesc() *prometheus.Desc {
	labels := []string{}
	labels = append(labels, c.sanitizedWorkloadLabels...)
	labels = append(labels, originBytesLabels...)

	return prometheus.NewDesc("conntrack_workload_reply_bytes_total", "Number of reply bytes", labels, nil)
}

func (c *ConntrackCollector) sendMetrics(counts map[accumulatorKey]int, workloads map[string]*workload.Workload, ch chan<- prometheus.Metric) {
	workloadConnectionsDesc := c.workloadConnectionsDesc()
	c.lastUsedWorkloadTuples.Range(func(key, _ interface{}) bool {
		accumulator := key.(accumulatorKey)
		count := counts[accumulator]
		workload := workloads[accumulator.workload]
		if workload == nil {
			return true
		}
		values := make([]string, c.connectionMetricTupleSize)

		values[0] = workload.Name
		i := 1
		for _, k := range c.workloadLabels {
			values[i] = workload.Labels[k]
			i++
		}
		values[i] = accumulator.state
		values[i+1] = accumulator.protocol
		values[i+2] = accumulator.destination.String()
		if accumulator.destination.ip != "" {
			values[i+3] = c.dnsCache.ResolveIP(accumulator.destination.ip)
		}
		values[i+4] = string(accumulator.direction)
		ch <- prometheus.MustNewConstMetric(workloadConnectionsDesc, prometheus.GaugeValue, float64(count), values...)
		return true
	})

	c.trafficCounter.RLock()
	defer c.trafficCounter.RUnlock()

	trafficBytesItems := c.trafficCounter.List()

	originBytesLabelDesc := c.workloadOriginBytesTotalDesc()
	for _, trafficBytesItem := range trafficBytesItems {
		workload := workloads[trafficBytesItem.Workload]

		if workload == nil {
			continue
		}

		ch <- prometheus.MustNewConstMetric(originBytesLabelDesc, prometheus.CounterValue, float64(trafficBytesItem.OriginCounter), c.bytesLabels(workload, trafficBytesItem.DestinationString())...)
	}

	replyBytesTotalDesc := c.workloadReplyBytesTotalDesc()

	for _, trafficBytesItem := range trafficBytesItems {
		workload := workloads[trafficBytesItem.Workload]

		if workload == nil {
			continue
		}

		ch <- prometheus.MustNewConstMetric(replyBytesTotalDesc, prometheus.CounterValue, float64(trafficBytesItem.ReplyCounter), c.bytesLabels(workload, trafficBytesItem.DestinationString())...)
	}
}

func (c *ConntrackCollector) bytesLabels(workload *workload.Workload, destination string) []string {
	values := make([]string, len(c.workloadLabels)+2)
	values[0] = workload.Name
	i := 1
	for _, k := range c.workloadLabels {
		values[i] = workload.Labels[k]
		i++
	}
	values[i] = destination

	return values
}
