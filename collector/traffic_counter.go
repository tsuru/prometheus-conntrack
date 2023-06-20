package collector

import (
	"fmt"
	"sync"
	"time"
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
	LastUsed      time.Time
}

type conntTrafficItem struct {
	connTrafficKey
	connTrafficValue
}

type trafficCounter struct {
	sync.RWMutex
	m                 map[connTrafficKey]*connTrafficValue
	previousConnState map[uint32]*connTrafficValue
}

func newTrafficCounter() *trafficCounter {
	tc := &trafficCounter{
		RWMutex:           sync.RWMutex{},
		m:                 make(map[connTrafficKey]*connTrafficValue),
		previousConnState: make(map[uint32]*connTrafficValue),
	}

	go tc.cleaner()
	return tc
}

func (t *trafficCounter) Inc(key connTrafficKey, id uint32, originCounter uint64, replyCounter uint64, now time.Time) {
	// TODO: clean conn ID that was used before to avoid colision
	v, ok := t.m[key]

	if !ok {
		v = &connTrafficValue{OriginCounter: 0, ReplyCounter: 0}
		t.m[key] = v
	}
	v.LastUsed = now

	previousConnState, ok := t.previousConnState[id]

	if ok {
		var diffOriginCounter uint64
		if originCounter > previousConnState.OriginCounter {
			diffOriginCounter = originCounter - previousConnState.OriginCounter
		}
		if diffOriginCounter > 0 {
			v.OriginCounter += diffOriginCounter
		}

		var diffReplyCounter uint64
		if replyCounter > previousConnState.ReplyCounter {
			diffReplyCounter = replyCounter - previousConnState.ReplyCounter
		}

		if diffReplyCounter > 0 {
			v.ReplyCounter += diffReplyCounter
		}
	} else {
		v.OriginCounter += originCounter
		v.ReplyCounter += replyCounter
	}

	t.previousConnState[id] = &connTrafficValue{OriginCounter: originCounter, ReplyCounter: replyCounter, LastUsed: now}
}

func (t *trafficCounter) cleaner() {
	for {
		t.doClean()
		time.Sleep(unusedConnectionTTL)
	}
}

func (t *trafficCounter) doClean() {
	t.Lock()
	defer t.Unlock()

	now := time.Now().UTC()

	for key, value := range t.m {
		if now.After(value.LastUsed.Add(unusedConnectionTTL)) {
			delete(t.m, key)
		}
	}

	for key, value := range t.previousConnState {
		if now.After(value.LastUsed.Add(unusedConnectionTTL)) {
			delete(t.previousConnState, key)
		}
	}
}

func (t *trafficCounter) List() []conntTrafficItem {
	l := make([]conntTrafficItem, 0)
	for key, value := range t.m {
		l = append(l, conntTrafficItem{key, *value})
	}

	return l
}
