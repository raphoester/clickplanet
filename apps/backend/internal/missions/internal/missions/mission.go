package missions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNoOrder = errors.New("a mission has no order")

type Take struct {
	Account  AccountID
	Flag     string
	Previous string
	Ground   string
	At       time.Time
	Position uint64
	Index    uint32
}

type TakeMark struct {
	Position uint64
	Index    uint32
}

func takeMarkPast(take Take) TakeMark {
	return TakeMark{Position: take.Position, Index: take.Index + 1}
}

func (m TakeMark) counted(take Take) bool {
	return take.Position < m.Position || take.Position == m.Position && take.Index < m.Index
}

type MissionID uuid.UUID

func (id MissionID) String() string { return uuid.UUID(id).String() }

type RewardKind string

const (
	RewardShields       RewardKind = "shields"
	RewardSpreadClicks  RewardKind = "spread_clicks"
	RewardBomb          RewardKind = "bomb"
	RewardEncloseClicks RewardKind = "enclose_clicks"
)

type Reward struct {
	Kind   RewardKind
	Amount uint32
}

// An order is one kind of mission: its own fields, and its own progress from 0. A new kind is a new type.
type Order interface {
	advanced(progress uint32, take Take) uint32
	done(progress uint32) bool
}

// Take Goal tiles for Flag.
type TakeFor struct {
	Flag string
	Goal uint32
}

func (o TakeFor) advanced(progress uint32, take Take) uint32 {
	return tileTaken(progress, o.Goal, take.Flag == o.Flag)
}
func (o TakeFor) done(progress uint32) bool { return progress >= o.Goal }

// Take Goal tiles that Flag held.
type TakeFrom struct {
	Flag string
	Goal uint32
}

func (o TakeFrom) advanced(progress uint32, take Take) uint32 {
	return tileTaken(progress, o.Goal, take.Previous == o.Flag)
}
func (o TakeFrom) done(progress uint32) bool { return progress >= o.Goal }

// Take Goal tiles on Ground, a country's land.
type TakeIn struct {
	Ground string
	Goal   uint32
}

func (o TakeIn) advanced(progress uint32, take Take) uint32 {
	return tileTaken(progress, o.Goal, take.Ground == o.Ground)
}
func (o TakeIn) done(progress uint32) bool { return progress >= o.Goal }

func tileTaken(progress, goal uint32, counts bool) uint32 {
	if counts && progress < goal {
		return progress + 1
	}
	return progress
}

type MissionSnapshot struct {
	ID     MissionID
	Order  Order
	Reward Reward
}

type Mission struct {
	id     MissionID
	order  Order
	reward Reward
}

func MissionOf(snapshot MissionSnapshot) (Mission, error) {
	if snapshot.Order == nil {
		return Mission{}, ErrNoOrder
	}
	return Mission{id: snapshot.ID, order: snapshot.Order, reward: snapshot.Reward}, nil
}

func (m Mission) Snapshot() MissionSnapshot {
	return MissionSnapshot{ID: m.id, Order: m.order, Reward: m.reward}
}

type Brief struct {
	missions []Mission
}

func BriefOf(snapshots []MissionSnapshot) (Brief, error) {
	brief := Brief{missions: make([]Mission, 0, len(snapshots))}
	for _, snapshot := range snapshots {
		mission, err := MissionOf(snapshot)
		if err != nil {
			return Brief{}, err
		}
		brief.missions = append(brief.missions, mission)
	}
	return brief, nil
}

func (b Brief) Snapshot() []MissionSnapshot {
	snapshots := make([]MissionSnapshot, 0, len(b.missions))
	for _, mission := range b.missions {
		snapshots = append(snapshots, mission.Snapshot())
	}
	return snapshots
}
