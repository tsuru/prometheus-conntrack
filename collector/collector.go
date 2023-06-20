// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tsuru/prometheus-conntrack/workload"

	promstrutil "github.com/prometheus/prometheus/util/strutil"
)

var (
	connectionLabels  = []string{"state", "protocol", "destination", "destination_name", "destination_zone", "direction"}
	originBytesLabels = []string{"destination", "destination_name", "destination_zone"}

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

	nodeIPs map[string]struct{}
	// lastUsedWorkloadTuples works such as a TTL, prometheus needs to know when connection is closed
	// then we will inform metric with 0 value for a while
	lastUsedWorkloadTuples sync.Map

	trafficCounter *trafficCounter
	cidrClassifier *cidrClassifier
}

func New(engine workload.Engine, conntrack Conntrack, workloadLabels []string, dnsCache DNSCache, classifier *cidrClassifier) (*ConntrackCollector, error) {
	sanitizedWorkloadLabels := []string{engine.Kind()}
	for _, workloadLabel := range workloadLabels {
		sanitizedWorkloadLabels = append(sanitizedWorkloadLabels, "label_"+promstrutil.SanitizeLabelName(workloadLabel))
	}

	if dnsCache == nil {
		dnsCache = newDNSCache()
	}

	ips, err := nodeIPs()
	if err != nil {
		return nil, err
	}

	for ip := range ips {
		fmt.Println("Found node IP:", ip)
	}

	collector := &ConntrackCollector{
		engine:                    engine,
		conntrack:                 conntrack,
		workloadLabels:            workloadLabels,
		sanitizedWorkloadLabels:   sanitizedWorkloadLabels,
		connectionMetricTupleSize: 1 + len(workloadLabels) + len(connectionLabels),
		dnsCache:                  dnsCache,
		nodeIPs:                   ips,
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
		cidrClassifier: classifier,
		trafficCounter: newTrafficCounter(),
	}

	go collector.metricCleaner()
	return collector, nil
}

func (c *ConntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.workloadConnectionsDesc()
	ch <- c.nodeConnectionsDesc()
	ch <- c.workloadOriginBytesTotalDesc()
	ch <- c.nodeOriginBytesTotalDesc()
	ch <- c.workloadReplyBytesTotalDesc()
	ch <- c.nodeReplyBytesTotalDesc()
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
	now := time.Now().UTC()

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

			c.trafficCounter.Inc(connTrafficKey{Workload: workload.Name, IP: d.ip, Port: d.port}, conn.ID, conn.OriginBytes, conn.ReplyBytes, now)
		}

		workloadMap[workload.Name] = workload
	}

	for _, conn := range conns {
		var d destination
		var direction ConnDirection

		if _, ok := c.nodeIPs[conn.OriginIP]; ok {
			d = destination{conn.DestIP, conn.DestPort}
			direction = OutgoingConnection
		} else if _, ok := c.nodeIPs[conn.DestIP]; ok {
			d = destination{"", conn.DestPort}
			direction = IncomingConnection
		} else {
			continue
		}

		key := accumulatorKey{
			protocol:    conn.Protocol,
			state:       conn.State,
			destination: d,
			direction:   direction,
		}
		counts[key] = counts[key] + 1

		c.trafficCounter.Inc(connTrafficKey{IP: d.ip, Port: d.port}, conn.ID, conn.OriginBytes, conn.ReplyBytes, now)
	}

	c.trafficCounter.Unlock()

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

func (c *ConntrackCollector) nodeConnectionsDesc() *prometheus.Desc {
	return prometheus.NewDesc("conntrack_node_connections", "Number of outbound node connections by destination and state", connectionLabels, nil)
}

func (c *ConntrackCollector) workloadConnectionsDesc() *prometheus.Desc {
	labels := []string{}
	labels = append(labels, c.sanitizedWorkloadLabels...)
	labels = append(labels, connectionLabels...)

	return prometheus.NewDesc("conntrack_workload_connections", "Number of outbound worload connections by destination and state", labels, nil)
}

func (c *ConntrackCollector) workloadOriginBytesTotalDesc() *prometheus.Desc {
	labels := []string{}
	labels = append(labels, c.sanitizedWorkloadLabels...)
	labels = append(labels, originBytesLabels...)

	return prometheus.NewDesc("conntrack_workload_origin_bytes_total", "Number of origin bytes", labels, nil)
}

func (c *ConntrackCollector) nodeOriginBytesTotalDesc() *prometheus.Desc {
	return prometheus.NewDesc("conntrack_node_origin_bytes_total", "Number of origin bytes", originBytesLabels, nil)
}

func (c *ConntrackCollector) workloadReplyBytesTotalDesc() *prometheus.Desc {
	labels := []string{}
	labels = append(labels, c.sanitizedWorkloadLabels...)
	labels = append(labels, originBytesLabels...)

	return prometheus.NewDesc("conntrack_workload_reply_bytes_total", "Number of reply bytes", labels, nil)
}

func (c *ConntrackCollector) nodeReplyBytesTotalDesc() *prometheus.Desc {
	return prometheus.NewDesc("conntrack_node_reply_bytes_total", "Number of reply bytes", originBytesLabels, nil)
}

func (c *ConntrackCollector) sendMetrics(counts map[accumulatorKey]int, workloads map[string]*workload.Workload, ch chan<- prometheus.Metric) {
	workloadConnectionsDesc := c.workloadConnectionsDesc()
	nodeConnectionsDesc := c.nodeConnectionsDesc()

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
			values[i+4] = c.cidrClassifier.Classify(accumulator.destination.ip)
		}

		values[i+5] = string(accumulator.direction)
		ch <- prometheus.MustNewConstMetric(workloadConnectionsDesc, prometheus.GaugeValue, float64(count), values...)
		return true
	})

	c.lastUsedWorkloadTuples.Range(func(key, _ interface{}) bool {
		accumulator := key.(accumulatorKey)
		count := counts[accumulator]
		if accumulator.workload != "" {
			return true
		}

		values := []string{
			accumulator.state,
			accumulator.protocol,
			accumulator.destination.String(),
			"",
			"",
			string(accumulator.direction),
		}

		if accumulator.destination.ip != "" {
			values[3] = c.dnsCache.ResolveIP(accumulator.destination.ip)
			values[4] = c.cidrClassifier.Classify(accumulator.destination.ip)
		}
		ch <- prometheus.MustNewConstMetric(nodeConnectionsDesc, prometheus.GaugeValue, float64(count), values...)
		return true
	})

	c.trafficCounter.RLock()
	defer c.trafficCounter.RUnlock()

	trafficBytesItems := c.trafficCounter.List()

	// workloads origin
	workloadOriginBytesLabelDesc := c.workloadOriginBytesTotalDesc()
	for _, trafficBytesItem := range trafficBytesItems {
		workload := workloads[trafficBytesItem.Workload]

		if workload == nil {
			continue
		}

		ch <- prometheus.MustNewConstMetric(workloadOriginBytesLabelDesc, prometheus.CounterValue, float64(trafficBytesItem.OriginCounter), c.workloadBytesLabels(workload, trafficBytesItem.connTrafficKey)...)
	}

	// nodes origin
	nodeOriginBytesLabelDesc := c.nodeOriginBytesTotalDesc()
	for _, trafficBytesItem := range trafficBytesItems {
		if trafficBytesItem.Workload != "" {
			continue
		}

		ch <- prometheus.MustNewConstMetric(nodeOriginBytesLabelDesc, prometheus.CounterValue, float64(trafficBytesItem.OriginCounter), c.destinationLabels(trafficBytesItem.connTrafficKey)...)
	}

	// workload reply
	replyBytesTotalDesc := c.workloadReplyBytesTotalDesc()
	for _, trafficBytesItem := range trafficBytesItems {
		workload := workloads[trafficBytesItem.Workload]

		if workload == nil {
			continue
		}

		ch <- prometheus.MustNewConstMetric(replyBytesTotalDesc, prometheus.CounterValue, float64(trafficBytesItem.ReplyCounter), c.workloadBytesLabels(workload, trafficBytesItem.connTrafficKey)...)
	}

	// node reply
	nodeReplyBytesTotalDesc := c.nodeReplyBytesTotalDesc()
	for _, trafficBytesItem := range trafficBytesItems {
		if trafficBytesItem.Workload != "" {
			continue
		}

		ch <- prometheus.MustNewConstMetric(nodeReplyBytesTotalDesc, prometheus.CounterValue, float64(trafficBytesItem.ReplyCounter), c.destinationLabels(trafficBytesItem.connTrafficKey)...)
	}
}

func (c *ConntrackCollector) workloadBytesLabels(workload *workload.Workload, destination connTrafficKey) []string {
	values := make([]string, len(c.workloadLabels)+4)
	values[0] = workload.Name
	i := 1
	for _, k := range c.workloadLabels {
		values[i] = workload.Labels[k]
		i++
	}
	values[i] = destination.DestinationString()
	if destination.IP == "" {
		values[i+1] = ""
		values[i+2] = ""
	} else {
		values[i+1] = c.dnsCache.ResolveIP(destination.IP)
		values[i+2] = c.cidrClassifier.Classify(destination.IP)
	}

	return values
}

func (c *ConntrackCollector) destinationLabels(destination connTrafficKey) []string {
	values := []string{
		destination.DestinationString(),
		"",
		"",
	}

	if destination.IP != "" {
		values[1] = c.dnsCache.ResolveIP(destination.IP)
		values[2] = c.cidrClassifier.Classify(destination.IP)
	}

	return values
}

func nodeIPs() (map[string]struct{}, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := map[string]struct{}{}

	for _, iface := range interfaces {
		if skipIface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			ip := strings.Split(addr.String(), "/")[0]

			if !skipIp(ip) {
				result[ip] = struct{}{}
			}
		}
	}

	return result, nil
}

var denyListNodeIPs = map[string]bool{"127.0.0.1": true, "::1": true}

func skipIp(ip string) bool {
	return denyListNodeIPs[ip] || strings.HasPrefix(ip, "169.254")
}

func skipIface(name string) bool {
	return strings.HasPrefix(name, "cali") || strings.HasPrefix(name, "docker") || name == "lo" || name == "nodelocaldns"
}
