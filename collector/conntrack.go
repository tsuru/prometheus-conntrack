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

// copied from: https://github.com/torvalds/linux/blob/master/include/uapi/linux/netfilter/nf_conntrack_tcp.h#L9
var (
	TCP_CONNTRACK_SYN_SENT    uint8 = 1
	TCP_CONNTRACK_ESTABLISHED uint8 = 3
	TCP_CONNTRACK_CLOSE_WAIT  uint8 = 5
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
		// TODO(wpjunior): track UDP connections
		if entry.ProtoInfo == nil || entry.ProtoInfo.TCP == nil {
			continue
		}
		var state string
		if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_ESTABLISHED {
			state = "ESTABLISHED"
		} else if *entry.ProtoInfo.TCP.State == TCP_CONNTRACK_SYN_SENT && entry.Timestamp != nil && entry.Timestamp.Start.Before(synSentDeadline) {
			state = "SYN-SENT"
		} else {
			continue
		}
		conns = append(conns, &Conn{
			OriginIP:   entry.Origin.Src.String(),
			OriginPort: port(entry.Origin.Proto.SrcPort),
			ReplyIP:    entry.Reply.Src.String(),
			ReplyPort:  port(entry.Reply.Proto.SrcPort),
			State:      state,
			Protocol:   "TCP",
		})
	}
	return conns
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
