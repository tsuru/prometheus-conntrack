// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"time"

	ct "github.com/florianl/go-conntrack"
	"github.com/pkg/errors"
)

var (
	// copied from: https://github.com/torvalds/linux/blob/master/include/uapi/linux/netfilter/nf_conntrack_tcp.h#L9
	TCP_CONNTRACK_SYN_SENT    uint8 = 1
	TCP_CONNTRACK_ESTABLISHED uint8 = 3
	TCP_CONNTRACK_CLOSE_WAIT  uint8 = 5
	TCP_CONNTRACK_LAST_ACK    uint8 = 6
	TCP_CONNTRACK_TIME_WAIT   uint8 = 7

	// copied from: https://github.com/torvalds/linux/blob/0d81a3f29c0afb18ba2b1275dcccf21e0dd4da38/include/uapi/linux/in.h#L28
	IPPROTO_TCP uint8 = 6
	IPPROTO_UDP uint8 = 17
)

var syncSentToleration = time.Second * 10

type Conn struct {
	ID          uint32
	OriginIP    string
	DestIP      string
	OriginPort  uint16
	DestPort    uint16
	State       string
	Protocol    string
	OriginBytes uint64
	ReplyBytes  uint64
}

func conntrack(protocol string) ([]*Conn, error) {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "Could not create nfct")
	}
	defer nfct.Close()
	entries, err := nfct.Dump(ct.Conntrack, ct.IPv4)
	if err != nil {
		return nil, errors.Wrap(err, "Could not dump conntrack entries")
	}

	return convertContrackEntryToConn(entries), nil
}

func convertContrackEntryToConn(entries []ct.Con) []*Conn {
	now := time.Now().UTC()
	conns := []*Conn{}
	synSentDeadline := now.Add(syncSentToleration * -1)
	for _, entry := range entries {

		var id uint32
		if entry.ID != nil {
			id = *entry.ID
		}

		var originBytes uint64
		if entry.CounterOrigin != nil {
			originBytes = *entry.CounterOrigin.Bytes
		}

		var replyBytes uint64
		if entry.CounterReply != nil {
			replyBytes = *entry.CounterReply.Bytes
		}

		proto, state := extractPROTOAndState(&entry, synSentDeadline)
		if state == "" {
			continue
		}
		conns = append(conns, &Conn{
			ID:          id,
			OriginIP:    entry.Origin.Src.String(),
			OriginPort:  port(entry.Origin.Proto.SrcPort),
			DestIP:      entry.Origin.Dst.String(),
			DestPort:    port(entry.Origin.Proto.DstPort),
			State:       state,
			OriginBytes: originBytes,
			ReplyBytes:  replyBytes,
			Protocol:    proto,
		})
	}
	return conns
}

func extractPROTOAndState(entry *ct.Con, synSentDeadline time.Time) (proto, state string) {
	if *entry.Origin.Proto.Number == IPPROTO_TCP && entry.ProtoInfo != nil && entry.ProtoInfo.TCP != nil {
		if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_ESTABLISHED {
			return "TCP", "ESTABLISHED"
		}
		if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_CLOSE_WAIT {
			return "TCP", "CLOSE-WAIT"
		}
		if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_TIME_WAIT {
			return "TCP", "TIME-WAIT"
		}
		if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_SYN_SENT && entry.Timestamp != nil && entry.Timestamp.Start.Before(synSentDeadline) {
			return "TCP", "SYN-SENT"
		}

		return "", ""
	}

	if *entry.Origin.Proto.Number == IPPROTO_UDP {
		return "UDP", "OPEN"
	}

	return "", ""
}

func NewConntrack(protocol string) Conntrack {
	return func() ([]*Conn, error) {
		return conntrack(protocol)
	}
}

func port(p *uint16) uint16 {
	if p == nil {
		return 0
	}

	return *p
}
