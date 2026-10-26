package rules

import (
	"sync/atomic"
)

type InForce struct {
	set atomic.Pointer[Set]
}

func NewInForce() *InForce {
	return &InForce{}
}

func (f *InForce) Enforce(set Set) {
	f.set.Store(&set)
}

func (f *InForce) Name() SetName { return f.set.Load().name }
