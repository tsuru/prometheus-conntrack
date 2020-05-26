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
const TCP_CONNTRACK_ESTABLISHED = 3

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
	sessions, err := nfct.Dump(ct.Conntrack, ct.IPv4)
	if err != nil {
		return nil, errors.Wrap(err, "Could not dump sessions")
	}
	conns := []*Conn{}
	for _, session := range sessions {
		// TODO(wpjunior): track UDP and SYN-SENT connections
		if session.ProtoInfo == nil || session.ProtoInfo.TCP == nil || *session.ProtoInfo.TCP.State != TCP_CONNTRACK_ESTABLISHED {
			continue
		}
		conns = append(conns, &Conn{
			OriginIP:   session.Origin.Src.String(),
			OriginPort: port(session.Origin.Proto.SrcPort),
			ReplyIP:    session.Reply.Src.String(),
			ReplyPort:  port(session.Reply.Proto.SrcPort),
			State:      "ESTABLISHED",
			Protocol:   "TCP",
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
