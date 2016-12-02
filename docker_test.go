// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/fsouza/go-dockerclient/testing"

	check "gopkg.in/check.v1"
)

func createContainer(c *check.C, url, name string) string {
	dockerClient, err := docker.NewClient(url)
	c.Assert(err, check.IsNil)
	err = dockerClient.PullImage(docker.PullImageOptions{Repository: "myimg"}, docker.AuthConfiguration{})
	c.Assert(err, check.IsNil)
	config := docker.Config{
		Image: "myimg",
		Cmd:   []string{"mycmd"},
	}
	opts := docker.CreateContainerOptions{Name: name, Config: &config}
	cont, err := dockerClient.CreateContainer(opts)
	c.Assert(err, check.IsNil)
	err = dockerClient.StartContainer(cont.ID, &docker.HostConfig{})
	c.Assert(err, check.IsNil)
	return cont.ID
}

func (s *S) TestInspect(c *check.C) {
	dockerServer, err := testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, check.IsNil)
	defer dockerServer.Stop()
	id := createContainer(c, dockerServer.URL(), "my-container")
	cont, err := inspect(id, dockerServer.URL())
	c.Assert(err, check.IsNil)
	c.Assert(cont.ID, check.Equals, id)
}

func (s *S) TestListContainers(c *check.C) {
	dockerServer, err := testing.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, check.IsNil)
	defer dockerServer.Stop()
	id := createContainer(c, dockerServer.URL(), "my-container")
	containers, err := listContainers(dockerServer.URL())
	c.Assert(err, check.IsNil)
	c.Assert(len(containers), check.Equals, 1)
	c.Assert(containers[0].ID, check.Equals, id)
}
