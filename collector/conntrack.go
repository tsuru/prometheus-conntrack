// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"strconv"

	ct "github.com/florianl/go-conntrack"
	"github.com/pkg/errors"
)

// TCP_CONNTRACK_ESTABLISHED copied from: https://github.com/torvalds/linux/blob/master/include/uapi/linux/netfilter/nf_conntrack_tcp.h#L9
var TCP_CONNTRACK_ESTABLISHED uint8 = 3

type Conn struct {
	SourceIP        string
	DestinationIP   string
	SourcePort      string
	DestinationPort string
	State           string
	Protocol        string
}

type conntrackResult struct {
	Items []struct {
		Metas []struct {
			SourceIP string `xml:"layer3>src"`
			DestIP   string `xml:"layer3>dst"`
			State    string `xml:"state"`
			Layer4   struct {
				SourcePort string `xml:"sport"`
				DestPort   string `xml:"dport"`
				Protocol   string `xml:"protoname,attr"`
			} `xml:"layer4"`
		} `xml:"meta"`
	} `xml:"flow"`
}

func conntrack(protocol string) ([]*Conn, error) {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "Could not create nfct")
	}
	defer nfct.Close()
	sessions, err := nfct.Dump(ct.Conntrack, ct.IPv4)
	if err != nil {
		return nil, errors.Wrap(err, "Could not dump sessions")
	}
	conns := []*Conn{}
	for _, session := range sessions {
		if session.ProtoInfo == nil || session.ProtoInfo.TCP == nil || *session.ProtoInfo.TCP.State != TCP_CONNTRACK_ESTABLISHED {
			continue
		}
		conns = append(conns, &Conn{
			SourceIP:        session.Origin.Src.String(),
			SourcePort:      port(session.Origin.Proto.SrcPort),
			DestinationIP:   session.Reply.Src.String(),
			DestinationPort: port(session.Reply.Proto.SrcPort),
			State:           "ESTABLISHED",
			Protocol:        "TCP",
		})
	}

	return conns, nil
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
