// Copyright 2020 conntrack-prometheus authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kubelet

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
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
	Opts

	client *http.Client
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
	response, err := k.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
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

type Opts struct {
	Endpoint string
	Key      string
	Cert     string
	CA       string
}

func NewEngine(opts Opts) (workload.Engine, error) {
	engine := &kubeletEngine{Opts: opts, client: http.DefaultClient}

	if opts.Key != "" && opts.Cert != "" {
		cert, err := tls.LoadX509KeyPair(opts.Cert, opts.Key)
		if err != nil {
			return nil, errors.Wrap(err, "could not read cert and key file")
		}

		caCert, err := ioutil.ReadFile(opts.CA)
		if err != nil {
			return nil, errors.Wrap(err, "could not read CA file")
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Setup HTTPS client
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
		tlsConfig.BuildNameToCertificate()
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		engine.client = &http.Client{Transport: transport}
	}

	return engine, nil
}
