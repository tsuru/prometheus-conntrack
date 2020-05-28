package collector

import (
	"net"
	"strings"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	cacheCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "conntrack_dns_cache_calls_total",
		Help: "The number of hits on cache of DNS",
	}, []string{"kind"})

	dnsErrorTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "conntrack_dns_error_total",
		Help: "The number of errors to call DNS",
	})
)

type DNSCache interface {
	ResolveIP(ip string) (addr string)
}

type dnsCache struct {
	c *cache.Cache
}

func newDNSCache() *dnsCache {
	return &dnsCache{c: cache.New(30*time.Minute, time.Minute)}
}

func (d *dnsCache) ResolveIP(ip string) string {
	addr, found := d.c.Get(ip)
	if found {
		cacheCallsTotal.WithLabelValues("hit").Inc()
		return addr.(string)
	}

	cacheCallsTotal.WithLabelValues("miss").Inc()

	names, err := net.LookupAddr(ip)
	if err != nil {
		dnsErrorTotal.Inc()
	}

	name := ""
	if len(names) > 0 {
		name = strings.TrimRight(names[0], ".")
	}

	d.c.Set(ip, name, cache.DefaultExpiration)
	return name
}
