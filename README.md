Prometheus Conntrack Metrics
============================

[![Build Status](https://travis-ci.org/tsuru/prometheus-conntrack.png?branch=master)](https://travis-ci.org/tsuru/prometheus-conntrack)

`prometheus-conntrack` exposes [conntrack](http://conntrack-tools.netfilter.org/) metrics for docker containers or kubelet pods.

For example, the series `conntrack_workload_connections{destination="192.168.50.4:2375",container="my-container",protocol="TCP",state="ESTABLISHED"} 2` means that the container "my-container" has two connections established with tcp://192.168.50.4:2375.

You can run `prometheus-conntrack` in a container or run the binary directly.

Docker Usage
--------------

```
$ go get github.com/tsuru/prometheus-conntrack
$ prometheus-conntrack --listen-address :8080 --docker-endpoint unix:///var/run/docker.sock
```

`prometheus-conntrack` will fetch running containers from the `--docker-endpoint` and
expose their outbound connections on `:8080/metrics`.


Kubelet Usage
---------------

```
$ go get github.com/tsuru/prometheus-conntrack
$ prometheus-conntrack -engine kubelet --listen-address :8080
```

`prometheus-conntrack` will fetch running pods from the local kubelet
