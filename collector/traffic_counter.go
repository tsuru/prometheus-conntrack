package collector

import (
	"fmt"
	"sync"
)

type connTrafficKey struct {
	Workload  string
	IP        string
	Port      uint16
	Direction ConnDirection
}

func (c connTrafficKey) DestinationString() string {
	return fmt.Sprintf("%s:%d", c.IP, c.Port)
}

type connTrafficValue struct {
	OriginCounter uint64
	ReplyCounter  uint64
}

type conntTrafficItem struct {
	connTrafficKey
	connTrafficValue
}

type trafficCounter struct {
	sync.RWMutex
	m map[connTrafficKey]*connTrafficValue

	previousConnState map[uint32]connTrafficValue
}

func newTrafficCounter() *trafficCounter {
	return &trafficCounter{
		RWMutex:           sync.RWMutex{},
		m:                 make(map[connTrafficKey]*connTrafficValue),
		previousConnState: make(map[uint32]connTrafficValue),
	}
}

func (t *trafficCounter) Inc(key connTrafficKey, id uint32, originCounter uint64, replyCounter uint64) {
	v, ok := t.m[key]

	if !ok {
		v = &connTrafficValue{}
		t.m[key] = v
	}

	previousConnState, ok := t.previousConnState[id]

	if ok {
		diffOriginCounter := originCounter - previousConnState.OriginCounter

		if diffOriginCounter > 0 {
			v.OriginCounter += diffOriginCounter
		}

		diffReplyCounter := replyCounter - previousConnState.ReplyCounter

		if diffReplyCounter > 0 {
			v.ReplyCounter += diffReplyCounter
		}
	} else {
		v.OriginCounter += originCounter
		v.ReplyCounter += replyCounter
	}

	t.previousConnState[id] = connTrafficValue{OriginCounter: originCounter, ReplyCounter: replyCounter}
}

func (t *trafficCounter) List() []conntTrafficItem {
	l := make([]conntTrafficItem, 0)
	for key, value := range t.m {
		l = append(l, conntTrafficItem{key, *value})
	}

	return l
}
