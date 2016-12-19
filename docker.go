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
	resp, err := client.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"status": {"running"},
		},
	})
	if err != nil {
		return nil, err
	}
	containers := make([]*docker.Container, len(resp))
	for i, c := range resp {
		container, err := client.InspectContainer(c.ID)
		if err != nil {
			return nil, err
		}
		containers[i] = container
	}
	return containers, nil
}
