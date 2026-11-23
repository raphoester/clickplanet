//go:build testing

package clicks

import "sync"

// SnareLog keeps each trap sprung with the player who paid for it, and takes nothing from a bank.
type SnareLog struct {
	mu      sync.Mutex
	springs []Spring
}

type Spring struct {
	Payer Payer
	Trap  TrapSnapshot
}

func NewSnareLog() *SnareLog {
	return &SnareLog{}
}

func (l *SnareLog) Spring(payer Payer, trap TrapSnapshot) float64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.springs = append(l.springs, Spring{Payer: payer, Trap: trap})
	return 0
}

func (l *SnareLog) Springs() []Spring {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]Spring(nil), l.springs...)
}
