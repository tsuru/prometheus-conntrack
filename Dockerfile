FROM alpine:3.2
RUN  apk update && apk add ca-certificates tzdata && rm -rf /var/cache/apk/*
ADD /bin/prometheus-conntrack /bin/prometheus-conntrack
ENTRYPOINT ["/bin/prometheus-conntrack"]
