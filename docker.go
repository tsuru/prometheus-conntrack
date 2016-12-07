// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import docker "github.com/fsouza/go-dockerclient"

func listContainers(endpoint string) ([]*docker.Container, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, err
	}
	var containers []*docker.Container
	for _, c := range resp {
		if c.State != "running" {
			continue
		}
		container, err := client.InspectContainer(c.ID)
		if err != nil {
			return nil, err
		}
		containers = append(containers, container)
	}
	return containers, nil
}
