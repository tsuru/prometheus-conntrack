// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os/exec"
)

type conn struct {
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

func conntrack(protocols string) ([]*conn, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("conntrack", "-L", "-o", "xml")
	if protocols != "" {
		cmd.Args = append(cmd.Args, "-p", protocols)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("conntrack failed: %s. Output: %s", err, stderr.String())
	}
	var result conntrackResult
	err = xml.Unmarshal(stdout.Bytes(), &result)
	if err != nil {
		return nil, err
	}
	var conns []*conn
	for _, item := range result.Items {
		if len(item.Metas) > 0 {
			if item.Metas[0].SourceIP != "127.0.0.1" && item.Metas[0].DestIP != "127.0.0.1" {
				conns = append(conns, &conn{
					SourceIP:        item.Metas[0].SourceIP,
					SourcePort:      item.Metas[0].Layer4.SourcePort,
					DestinationIP:   item.Metas[0].DestIP,
					DestinationPort: item.Metas[0].Layer4.DestPort,
					State:           item.Metas[2].State,
					Protocol:        item.Metas[0].Layer4.Protocol,
				})
			}
		}
	}
	return conns, nil
}
