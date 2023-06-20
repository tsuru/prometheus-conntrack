FROM golang:1.19.10-buster as builder

COPY . /prometheus-conntrack
WORKDIR /prometheus-conntrack
RUN go build -ldflags "-linkmode external -extldflags -static" -o /bin/prometheus-conntrack

FROM scratch
COPY --from=builder /bin/prometheus-conntrack /bin/prometheus-conntrack
ENTRYPOINT ["/bin/prometheus-conntrack"]
