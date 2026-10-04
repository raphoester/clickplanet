package roster_query

import (
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

type Lines interface {
	Lines() []*playerv1.RosterEntry
}

func NewMemoryQuery(lines Lines) *MemoryQuery {
	return &MemoryQuery{lines: lines}
}

type MemoryQuery struct {
	lines Lines
}

func (q *MemoryQuery) Roster() *playerv1.GetRosterResponse {
	return &playerv1.GetRosterResponse{Entries: q.lines.Lines()}
}
