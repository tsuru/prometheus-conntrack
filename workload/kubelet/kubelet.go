// Copyright 2020 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kubelet

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tsuru/prometheus-conntrack/workload"
)

type podList struct {
	Items []pod `json:"items"`
}
type pod struct {
	Metadata podMetadata `json:"metadata"`
	Status   podStatus   `json:"status"`
}

type podMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type podStatus struct {
	PodIP string `json:"podIP"`
}

type kubeletEngine struct {
	endpoint string
}

func (d *kubeletEngine) Name() string {
	return "kubernetes"
}

func (d *kubeletEngine) Kind() string {
	return "pod"
}

func (k *kubeletEngine) Workloads() ([]*workload.Workload, error) {
	workloads := []*workload.Workload{}

	// TODO(wpjunior): add kubernetes authentication
	req, _ := http.NewRequest(http.MethodGet, k.endpoint, nil)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Invalid response code")
	}

	list := &podList{}
	err = json.NewDecoder(response.Body).Decode(&list)
	if err != nil {
		return nil, err
	}

	for _, pod := range list.Items {
		pod.Metadata.Labels["pod_namespace"] = pod.Metadata.Namespace
		workloads = append(workloads, &workload.Workload{
			Name:   pod.Metadata.Name,
			IP:     pod.Status.PodIP,
			Labels: pod.Metadata.Labels,
		})
	}

	return workloads, nil
}

func NewEngine(endpoint string) workload.Engine {
	return &kubeletEngine{endpoint: endpoint}
}
