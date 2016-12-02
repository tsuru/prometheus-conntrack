FROM alpine:3.2
ADD /bin/prometheus-conntrack /bin/prometheus-conntrack
ENTRYPOINT ["/bin/prometheus-conntrack"]
