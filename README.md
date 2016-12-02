Prometheus Conntrack Metrics
============================

`prometheus-conntrack` exposes [conntrack](http://conntrack-tools.netfilter.org/) metrics for docker containers.

For example, the series `container_connections{container_label_label1="val1",destination="192.168.50.4:2375",id="id",image="image",name="my-container",protocol="tcp",state="ESTABLISHED"} 2` means that the container "my-container" has two connections established with tcp://192.168.50.4:2375.

Usage
-----

You can run `prometheus-conntrack` in a container or run the binary directly.

```
$ go get github.com/tsuru/prometheus-conntrack
$ prometheus-conntrack --listen-address :8080 --docker-endpoint unix:///var/run/docker.sock
```

`prometheus-conntrack` will fetch running containers from the `--docker-endpoint` and
expose their outbound connections on `:8080/metrics`.
