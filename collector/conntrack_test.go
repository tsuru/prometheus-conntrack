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
			Origin:    &ct.IPTuple{Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP}},
			Reply:     &ct.IPTuple{Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP}},
			ProtoInfo: nil,
		},
		{
			Origin:    &ct.IPTuple{Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP}},
			Reply:     &ct.IPTuple{Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP}},
			ProtoInfo: &ct.ProtoInfo{TCP: nil},
		},
		{
			Origin:    &ct.IPTuple{Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP}},
			Reply:     &ct.IPTuple{Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP}},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_CLOSE_WAIT}},
		},
		{
			Origin:    &ct.IPTuple{Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8080)}},
			Reply:     &ct.IPTuple{Src: parseIP("192.0.2.2"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8081)}},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_ESTABLISHED}},
		},
		{
			Origin:    &ct.IPTuple{Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8080)}},
			Reply:     &ct.IPTuple{Src: parseIP("192.0.2.4"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8081)}},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_SYN_SENT}},
			Timestamp: &ct.Timestamp{Start: &now},
		},
		{
			Origin:    &ct.IPTuple{Src: parseIP("192.0.2.1"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8080)}},
			Reply:     &ct.IPTuple{Src: parseIP("192.0.2.3"), Proto: &ct.ProtoTuple{Number: &IPPROTO_TCP, SrcPort: portPtr(8081)}},
			ProtoInfo: &ct.ProtoInfo{TCP: &ct.TCPInfo{State: &TCP_CONNTRACK_SYN_SENT}},
			Timestamp: &ct.Timestamp{Start: &delayedConnStart},
		},
		{
			Origin: &ct.IPTuple{Src: parseIP("192.0.2.50"), Proto: &ct.ProtoTuple{Number: &IPPROTO_UDP, SrcPort: portPtr(8080)}},
			Reply:  &ct.IPTuple{Src: parseIP("192.0.2.51"), Proto: &ct.ProtoTuple{Number: &IPPROTO_UDP, SrcPort: portPtr(8081)}},
		},
	}
	conns := convertContrackEntryToConn(ctConn)

	assert.Equal(t, []*Conn{
		{OriginIP: "192.0.2.1", ReplyIP: "192.0.2.2", State: "ESTABLISHED", Protocol: "TCP", OriginPort: "8080", ReplyPort: "8081"},
		{OriginIP: "192.0.2.1", ReplyIP: "192.0.2.3", State: "SYN-SENT", Protocol: "TCP", OriginPort: "8080", ReplyPort: "8081"},
		{OriginIP: "192.0.2.50", ReplyIP: "192.0.2.51", State: "CONNECTED", Protocol: "UDP", OriginPort: "8080", ReplyPort: "8081"},
	}, conns)
}

func parseIP(ip string) *net.IP {
	fullIP := net.ParseIP(ip)
	return &fullIP
}

func portPtr(port uint16) *uint16 {
	return &port
}
