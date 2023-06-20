// Copyright 2020 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kubelet

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/tsuru/prometheus-conntrack/workload"
)

type podList struct {
	Items []pod `json:"items"`
}
type pod struct {
	Metadata podMetadata `json:"metadata"`
	Spec     podSpec     `json:"spec"`
	Status   podStatus   `json:"status"`
}

type podMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type podSpec struct {
	HostNetwork bool `json:"hostNetwork"`
}

type podStatus struct {
	PodIP string `json:"podIP"`
}

type kubeletEngine struct {
	Opts

	client       *http.Client
	tokenContent string
}

func (d *kubeletEngine) Name() string {
	return "kubernetes"
}

func (d *kubeletEngine) Kind() string {
	return "pod"
}

func (k *kubeletEngine) Workloads() ([]*workload.Workload, error) {
	workloads := []*workload.Workload{}

	req, _ := http.NewRequest(http.MethodGet, k.Endpoint, nil)
	if k.tokenContent != "" {
		req.Header.Set("Authorization", "Bearer "+k.tokenContent)
	}
	response, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Invalid response code: %d", response.StatusCode)
	}

	list := &podList{}
	err = json.NewDecoder(response.Body).Decode(&list)
	if err != nil {
		return nil, err
	}

	for _, pod := range list.Items {
		// we skip all pods with hostNetwork because its use the same ip of host
		// and may generate a mess in the metrics
		if pod.Spec.HostNetwork {
			continue
		}
		if pod.Metadata.Labels == nil {
			pod.Metadata.Labels = map[string]string{}
		}
		pod.Metadata.Labels["pod_namespace"] = pod.Metadata.Namespace
		workloads = append(workloads, &workload.Workload{
			Name:   pod.Metadata.Name,
			IP:     pod.Status.PodIP,
			Labels: pod.Metadata.Labels,
		})
	}

	return workloads, nil
}

type Opts struct {
	Endpoint string
	Key      string
	Cert     string
	CA       string
	Token    string

	InsecureSkipVerify bool
}

func NewEngine(opts Opts) (workload.Engine, error) {
	// Setup HTTPS client
	tlsConfig := &tls.Config{
		InsecureSkipVerify: opts.InsecureSkipVerify,
	}

	var tokenContent string
	if opts.Token != "" {
		tokenBytes, err := os.ReadFile(opts.Token)
		if err != nil {
			return nil, errors.Wrap(err, "could not read Token file")
		}
		tokenContent = string(tokenBytes)
	}
	if opts.CA != "" {
		caCert, err := os.ReadFile(opts.CA)
		if err != nil {
			return nil, errors.Wrap(err, "could not read CA file")
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	if opts.Key != "" && opts.Cert != "" {
		cert, err := tls.LoadX509KeyPair(opts.Cert, opts.Key)
		if err != nil {
			return nil, errors.Wrap(err, "could not read cert and key file")
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	return &kubeletEngine{Opts: opts, client: client, tokenContent: tokenContent}, nil
}
