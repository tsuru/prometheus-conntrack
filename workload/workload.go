// Copyright 2020 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workload

type Workload struct {
	Name   string
	IP     string
	Labels map[string]string
}

type Engine interface {
	Name() string
	Kind() string
	Workloads() ([]*Workload, error)
}
