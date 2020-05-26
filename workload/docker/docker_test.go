// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	dockerTesting "github.com/fsouza/go-dockerclient/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createContainer(t *testing.T, url, name string) string {
	dockerClient, err := docker.NewClient(url)
	require.NoError(t, err)
	err = dockerClient.PullImage(docker.PullImageOptions{Repository: "myimg"}, docker.AuthConfiguration{})
	require.NoError(t, err)
	config := docker.Config{
		Image: "myimg",
		Cmd:   []string{"mycmd"},
	}
	opts := docker.CreateContainerOptions{Name: name, Config: &config}
	cont, err := dockerClient.CreateContainer(opts)
	require.NoError(t, err)
	err = dockerClient.StartContainer(cont.ID, &docker.HostConfig{})
	require.NoError(t, err)
	return cont.ID
}

func TestListWorkloads(t *testing.T) {
	dockerServer, err := dockerTesting.NewServer("127.0.0.1:0", nil, nil)
	require.NoError(t, err)
	defer dockerServer.Stop()
	createContainer(t, dockerServer.URL(), "my-container")
	engine := NewEngine(dockerServer.URL())
	workloads, err := engine.Workloads()
	require.NoError(t, err)
	assert.Len(t, workloads, 1)
	assert.Equal(t, "my-container", workloads[0].Name)
	assert.Equal(t, "172.16.42.53", workloads[0].IP)
}
