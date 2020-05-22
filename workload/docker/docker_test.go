// Copyright 2016 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	dockerTesting "github.com/fsouza/go-dockerclient/testing"

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

var _ = check.Suite(&S{})

func Test(t *testing.T) {
	check.TestingT(t)
}

type S struct{}

func (s *S) TestListWorkloads(c *check.C) {
	dockerServer, err := dockerTesting.NewServer("127.0.0.1:0", nil, nil)
	c.Assert(err, check.IsNil)
	defer dockerServer.Stop()
	createContainer(c, dockerServer.URL(), "my-container")
	engine := NewEngine(dockerServer.URL())
	workloads, err := engine.Workloads()
	c.Assert(err, check.IsNil)
	c.Assert(len(workloads), check.Equals, 1)
	c.Assert(workloads[0].Name, check.Equals, "my-container")
	c.Assert(workloads[0].IP, check.Equals, "172.16.42.53")
}
