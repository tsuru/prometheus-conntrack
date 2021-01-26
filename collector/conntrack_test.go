// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"net"
	"testing"
	"time"

	ct "github.com/florianl/go-conntrack"
	"github.com/stretchr/testify/assert"
)

func TestConvertContrackEntryToConn(t *testing.T) {
	now := time.Now().UTC()
	delayedConnStart := now.Add(time.Minute * -1)

	ctConn := []ct.Con{
		{
			Origin: &ct.IPTuple{
				Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP},
				Dst: parseIP("192.0.2.2"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP},
				Dst: parseIP("192.0.2.1"),
			},
			ProtoInfo: nil,
		},
		{
			Origin: &ct.IPTuple{
				Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP},
				Dst: parseIP("192.0.2.2"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP},
				Dst: parseIP("192.0.2.1"),
			},
			ProtoInfo: &ct.ProtoInfo{TCP: nil},
		},
		{
			Origin: &ct.IPTuple{
				Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP},
				Dst: parseIP("192.0.2.2"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP},
				Dst: parseIP("192.0.2.1"),
			},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_CLOSE_WAIT}},
		},
		{
			Origin: &ct.IPTuple{
				Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8080), DstPort: portPtr(8081)},
				Dst: parseIP("192.0.2.2"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8081)},
				Dst: parseIP("192.0.2.1"),
			},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_ESTABLISHED}},
		},
		{
			Origin: &ct.IPTuple{
				Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8080), DstPort: portPtr(8081)},
				Dst: parseIP("192.0.2.4"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.4"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8081)},
				Dst: parseIP("192.0.2.1"),
			},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_SYN_SENT}},
			Timestamp: &ct.Timestamp{Start: &now},
		},
		{
			Origin: &ct.IPTuple{
				Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8080), DstPort: portPtr(8081)},
				Dst: parseIP("192.0.2.3"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.3"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8081)},
				Dst: parseIP("192.0.2.1"),
			},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_SYN_SENT}},
			Timestamp: &ct.Timestamp{Start: &delayedConnStart},
		},
		{
			Origin: &ct.IPTuple{
				Src: parseIP("192.0.2.50"), Proto: &ct.ProtoTuple{Number: &IPPROTO_UDP, SrcPort: portPtr(8080), DstPort: portPtr(8081)},
				Dst: parseIP("192.0.2.51"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.51"), Proto: &ct.ProtoTuple{Number: &IPPROTO_UDP, SrcPort: portPtr(8081)},
				Dst: parseIP("192.0.2.50"),
			},
		},
		{
			Origin: &ct.IPTuple{
				Src:   parseIP("192.0.2.1"),
				Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8080), DstPort: portPtr(8081)},
				Dst:   parseIP("172.68.0.1"),
			},
			Reply: &ct.IPTuple{
				Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8081)},
			},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_ESTABLISHED}},
		},
	}
	conns := convertContrackEntryToConn(ctConn)

	assert.Equal(t, []*Conn{
		{OriginIP: "192.0.2.1", DestIP: "192.0.2.2", State: "ESTABLISHED", Protocol: "TCP", OriginPort: "8080", DestPort: "8081"},
		{OriginIP: "192.0.2.1", DestIP: "192.0.2.3", State: "SYN-SENT", Protocol: "TCP", OriginPort: "8080", DestPort: "8081"},
		{OriginIP: "192.0.2.50", DestIP: "192.0.2.51", State: "OPEN", Protocol: "UDP", OriginPort: "8080", DestPort: "8081"},
		{OriginIP: "192.0.2.1", DestIP: "172.68.0.1", State: "ESTABLISHED", Protocol: "TCP", OriginPort: "8080", DestPort: "8081"},
	}, conns)
}

func parseIP(ip string) *net.IP {
	fullIP := net.ParseIP(ip)
	return &fullIP
}

func portPtr(port uint16) *uint16 {
	return &port
}
