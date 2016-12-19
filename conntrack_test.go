// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"

	"github.com/tsuru/commandmocker"
	"gopkg.in/check.v1"
)

var _ = check.Suite(&S{})

func Test(t *testing.T) {
	check.TestingT(t)
}

type S struct{}

func (*S) TestConntrack(c *check.C) {
	dir, err := commandmocker.Add("conntrack", conntrackXML)
	c.Assert(err, check.IsNil)
	defer commandmocker.Remove(dir)
	conns, err := conntrack()
	c.Assert(err, check.IsNil)
	expected := []*conn{
		{SourceIP: "192.168.50.4", SourcePort: "33404", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "172.17.42.1", SourcePort: "42418", DestinationIP: "172.17.0.2", DestinationPort: "4001", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "172.17.42.1", SourcePort: "42428", DestinationIP: "172.17.0.2", DestinationPort: "4001", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "192.168.50.4", SourcePort: "53922", DestinationIP: "192.168.50.4", DestinationPort: "5000", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "192.168.50.4", SourcePort: "43227", DestinationIP: "192.168.50.4", DestinationPort: "8080", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "172.17.0.27", SourcePort: "39502", DestinationIP: "172.17.42.1", DestinationPort: "4001", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "192.168.50.4", SourcePort: "33496", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "192.168.50.4", SourcePort: "33495", DestinationIP: "192.168.50.4", DestinationPort: "2375", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.211.55.2", SourcePort: "51388", DestinationIP: "10.211.55.184", DestinationPort: "22", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "172.17.0.27", SourcePort: "39492", DestinationIP: "172.17.42.1", DestinationPort: "4001", State: "ESTABLISHED", Protocol: "tcp"},
		{SourceIP: "10.211.55.2", SourcePort: "51370", DestinationIP: "10.211.55.184", DestinationPort: "22", State: "ESTABLISHED", Protocol: "tcp"},
	}
	c.Assert(conns, check.DeepEquals, expected)
}

func (*S) TestConntrackCommandFailure(c *check.C) {
	dir, err := commandmocker.Error("conntrack", "something went wrong", 120)
	c.Assert(err, check.IsNil)
	defer commandmocker.Remove(dir)
	conns, err := conntrack()
	c.Assert(err, check.ErrorMatches, "conntrack failed: exit status 120. Output: something went wrong")
	c.Assert(conns, check.IsNil)
}

const conntrackXML = `<?xml version="1.0" encoding="utf-8"?>
<conntrack>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>33404</sport><dport>2375</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>2375</sport><dport>33404</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431975</timeout><mark>0</mark><use>1</use><id>907489792</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>172.17.42.1</src><dst>172.17.0.2</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>42418</sport><dport>4001</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>172.17.0.2</src><dst>172.17.42.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>4001</sport><dport>42418</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431972</timeout><mark>0</mark><use>1</use><id>907492032</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>56823</sport><dport>27017</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>27017</sport><dport>56823</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431960</timeout><mark>0</mark><use>1</use><id>106406016</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>172.17.42.1</src><dst>172.17.0.2</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>42428</sport><dport>4001</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>172.17.0.2</src><dst>172.17.42.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>4001</sport><dport>42428</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431972</timeout><mark>0</mark><use>1</use><id>907492672</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>54495</sport><dport>27017</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>27017</sport><dport>54495</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431992</timeout><mark>0</mark><use>1</use><id>994211584</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>53922</sport><dport>5000</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>172.17.0.1</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>5000</sport><dport>53922</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431985</timeout><mark>0</mark><use>1</use><id>907490432</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>43227</sport><dport>8080</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>8080</sport><dport>43227</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431975</timeout><mark>0</mark><use>1</use><id>106408576</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>172.17.0.27</src><dst>172.17.42.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>39502</sport><dport>4001</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>172.17.42.1</src><dst>172.17.0.27</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>4001</sport><dport>39502</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431999</timeout><mark>0</mark><use>1</use><id>907491712</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>56073</sport><dport>27017</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>27017</sport><dport>56073</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431999</timeout><mark>0</mark><use>1</use><id>106356224</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>33496</sport><dport>2375</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>2375</sport><dport>33496</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>299</timeout><mark>0</mark><use>1</use><id>106400576</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>56753</sport><dport>27017</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>27017</sport><dport>56753</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431995</timeout><mark>0</mark><use>1</use><id>907490112</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>56752</sport><dport>27017</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>27017</sport><dport>56752</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431999</timeout><mark>0</mark><use>1</use><id>907492352</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>33495</sport><dport>2375</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>192.168.50.4</src><dst>192.168.50.4</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>2375</sport><dport>33495</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431196</timeout><mark>0</mark><use>1</use><id>106405376</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>10.211.55.2</src><dst>10.211.55.184</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>51388</sport><dport>22</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>10.211.55.184</src><dst>10.211.55.2</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>22</sport><dport>51388</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>299</timeout><mark>0</mark><use>1</use><id>106358464</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>172.17.0.27</src><dst>172.17.42.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>39492</sport><dport>4001</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>172.17.42.1</src><dst>172.17.0.27</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>4001</sport><dport>39492</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431999</timeout><mark>0</mark><use>1</use><id>907491392</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>10.211.55.2</src><dst>10.211.55.184</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>51370</sport><dport>22</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>10.211.55.184</src><dst>10.211.55.2</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>22</sport><dport>51370</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>430417</timeout><mark>0</mark><use>1</use><id>907488832</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>56754</sport><dport>27017</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>27017</sport><dport>56754</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431995</timeout><mark>0</mark><use>2</use><id>907485632</id><assured/></meta></flow>
<flow><meta direction="original"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>54483</sport><dport>27017</dport></layer4></meta><meta direction="reply"><layer3 protonum="2" protoname="ipv4"><src>127.0.0.1</src><dst>127.0.0.1</dst></layer3><layer4 protonum="6" protoname="tcp"><sport>27017</sport><dport>54483</dport></layer4></meta><meta direction="independent"><state>ESTABLISHED</state><timeout>431946</timeout><mark>0</mark><use>1</use><id>994198784</id><assured/></meta></flow>
</conntrack>`
