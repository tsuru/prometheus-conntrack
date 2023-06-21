package collector

import (
	"net"
	"sort"
)

type cidrClassifier struct {
	matchs []cidrMatch
	cache  map[string]string
}

type cidrMatch struct {
	cidr  net.IPNet
	label string
}

func (c *cidrClassifier) Classify(IP string) string {
	if cachedValue, ok := c.cache[IP]; ok {
		return cachedValue
	}

	parsedIP := net.ParseIP(IP)

	if parsedIP == nil {
		return ""
	}

	for _, m := range c.matchs {
		if m.cidr.Contains(parsedIP) {
			c.cache[IP] = m.label
			return m.label
		}
	}

	return ""
}

func NewCIDRClassifier(cidrsLabels map[string]string) (*cidrClassifier, error) {
	matchs := []cidrMatch{}
	for cidr, label := range cidrsLabels {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}

		matchs = append(matchs, cidrMatch{*ipNet, label})
	}

	sort.Slice(matchs, func(i, j int) bool {
		iBits, _ := matchs[i].cidr.Mask.Size()
		jBits, _ := matchs[j].cidr.Mask.Size()

		return jBits < iBits
	})

	return &cidrClassifier{
		matchs: matchs,
		cache:  map[string]string{},
	}, nil
}
