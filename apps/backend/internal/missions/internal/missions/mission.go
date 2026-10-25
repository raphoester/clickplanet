package missions

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNoSlot      = errors.New("no mission has that slot")
	ErrUnknownKind = errors.New("not a kind of mission")
)

type Take struct {
	Account  AccountID
	Flag     string
	Previous string
	Ground   string
	At       time.Time
	Position uint64
}

type Slot uint8

var Slots = []Slot{1, 2, 3}

func SlotOf(n uint32) (Slot, error) {
	for _, slot := range Slots {
		if uint32(slot) == n {
			return slot, nil
		}
	}
	return 0, fmt.Errorf("%w: %d", ErrNoSlot, n)
}

type Kind string

const (
	KindTakeFor  Kind = "take_for"
	KindTakeFrom Kind = "take_from"
	KindTakeIn   Kind = "take_in"
)

type RewardKind string

const (
	RewardShields      RewardKind = "shields"
	RewardSpreadClicks RewardKind = "spread_clicks"
	RewardBomb         RewardKind = "bomb"
)

type Reward struct {
	Kind   RewardKind
	Amount uint32
}

type MissionSnapshot struct {
	Slot   Slot
	Kind   Kind
	Target string
	Goal   uint32
	Reward Reward
}

type Mission struct {
	slot   Slot
	order  order
	goal   uint32
	reward Reward
}

type order interface {
	kind() Kind
	target() string
	counts(take Take) bool
}

type takeFor struct{ flag string }

func (o takeFor) kind() Kind            { return KindTakeFor }
func (o takeFor) target() string        { return o.flag }
func (o takeFor) counts(take Take) bool { return take.Flag == o.flag }

type takeFrom struct{ flag string }

func (o takeFrom) kind() Kind            { return KindTakeFrom }
func (o takeFrom) target() string        { return o.flag }
func (o takeFrom) counts(take Take) bool { return take.Previous == o.flag }

type takeIn struct{ ground string }

func (o takeIn) kind() Kind            { return KindTakeIn }
func (o takeIn) target() string        { return o.ground }
func (o takeIn) counts(take Take) bool { return take.Ground == o.ground }

func MissionOf(snapshot MissionSnapshot) (Mission, error) {
	var rule order
	switch snapshot.Kind {
	case KindTakeFor:
		rule = takeFor{flag: snapshot.Target}
	case KindTakeFrom:
		rule = takeFrom{flag: snapshot.Target}
	case KindTakeIn:
		rule = takeIn{ground: snapshot.Target}
	default:
		return Mission{}, fmt.Errorf("%w: %q", ErrUnknownKind, snapshot.Kind)
	}
	return Mission{slot: snapshot.Slot, order: rule, goal: snapshot.Goal, reward: snapshot.Reward}, nil
}

func (m Mission) Counts(take Take) bool { return m.order.counts(take) }

func (m Mission) Snapshot() MissionSnapshot {
	return MissionSnapshot{Slot: m.slot, Kind: m.order.kind(), Target: m.order.target(), Goal: m.goal, Reward: m.reward}
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

func (b Brief) Mission(slot Slot) (Mission, error) {
	for _, mission := range b.missions {
		if mission.slot == slot {
			return mission, nil
		}
	}
	return Mission{}, fmt.Errorf("%w: %d", ErrNoSlot, slot)
}

func (b Brief) Snapshot() []MissionSnapshot {
	snapshots := make([]MissionSnapshot, 0, len(b.missions))
	for _, mission := range b.missions {
		snapshots = append(snapshots, mission.Snapshot())
	}
	return snapshots
}
