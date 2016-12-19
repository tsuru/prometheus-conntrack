# Copyright 2016 conntrack-prometheus authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

build:
	go build -ldflags "-linkmode external -extldflags -static" -o bin/prometheus-conntrack

test:
	go test ./...

run: build
	./bin/prometheus-conntrack

bench:
	go test -check.b -check.bmem
