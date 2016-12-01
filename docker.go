// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import docker "github.com/fsouza/go-dockerclient"

func inspect(contID, endpoint string) (*docker.Container, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	return client.InspectContainer(contID)
}

func listContainers(endpoint string) ([]docker.APIContainers, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
