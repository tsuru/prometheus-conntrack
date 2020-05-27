// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"strconv"
	"time"

	ct "github.com/florianl/go-conntrack"
	"github.com/pkg/errors"
)

var (
	// copied from: https://github.com/torvalds/linux/blob/master/include/uapi/linux/netfilter/nf_conntrack_tcp.h#L9
	TCP_CONNTRACK_SYN_SENT    uint8 = 1
	TCP_CONNTRACK_ESTABLISHED uint8 = 3
	TCP_CONNTRACK_CLOSE_WAIT  uint8 = 5

	// copied from: https://github.com/torvalds/linux/blob/0d81a3f29c0afb18ba2b1275dcccf21e0dd4da38/include/uapi/linux/in.h#L28
	IPPROTO_TCP uint8 = 6
	IPPROTO_UDP uint8 = 17
)

var syncSentToleration = time.Second * 10

type Conn struct {
	OriginIP   string
	ReplyIP    string
	OriginPort string
	ReplyPort  string
	State      string
	Protocol   string
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
		proto, state := extractPROTOAndState(&entry, synSentDeadline)
		if state == "" {
			continue
		}
		conns = append(conns, &Conn{
			OriginIP:   entry.Origin.Src.String(),
			OriginPort: port(entry.Origin.Proto.SrcPort),
			ReplyIP:    entry.Reply.Src.String(),
			ReplyPort:  port(entry.Reply.Proto.SrcPort),
			State:      state,
			Protocol:   proto,
		})
	}
	return conns
}

func extractPROTOAndState(entry *ct.Con, synSentDeadline time.Time) (proto, state string) {
	if *entry.Origin.Proto.Number == IPPROTO_TCP && entry.ProtoInfo != nil && entry.ProtoInfo.TCP != nil {
		if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_ESTABLISHED {
			return "TCP", "ESTABLISHED"
		}
		if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_SYN_SENT && entry.Timestamp != nil && entry.Timestamp.Start.Before(synSentDeadline) {
			return "TCP", "SYN-SENT"
		}

		return "", ""
	}

	if *entry.Origin.Proto.Number == IPPROTO_UDP {
		return "UDP", "CONNECTED"
	}

	return "", ""
}

func NewConntrack(protocol string) Conntrack {
	return func() ([]*Conn, error) {
		return conntrack(protocol)
	}
}

func port(p *uint16) string {
	if p == nil {
		return ""
	}

	return strconv.Itoa(int(*p))
}
