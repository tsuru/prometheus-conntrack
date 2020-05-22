// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	dockerClient "github.com/fsouza/go-dockerclient"
	"github.com/tsuru/prometheus-conntrack/workload"
)

type dockerContainerEngine struct {
	endpoint string
}

func (d *dockerContainerEngine) Name() string {
	return "docker"
}

func (d *dockerContainerEngine) Kind() string {
	return "container"
}

func (d *dockerContainerEngine) Workloads() ([]*workload.Workload, error) {
	workloads := []*workload.Workload{}
	client, err := dockerClient.NewClient(d.endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListContainers(dockerClient.ListContainersOptions{
		Filters: map[string][]string{
			"status": {"running"},
		},
	})
	if err != nil {
		return nil, err
	}
	for _, c := range resp {
		container, err := client.InspectContainer(c.ID)
		if err != nil {
			return nil, err
		}
		workloads = append(workloads, &workload.Workload{
			Name: container.Name,
			IP:   container.NetworkSettings.IPAddress,
		})
	}
	return workloads, nil
}

func NewEngine(endpoint string) workload.Engine {
	return &dockerContainerEngine{endpoint: endpoint}
}
