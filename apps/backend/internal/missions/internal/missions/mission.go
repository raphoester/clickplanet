package missions

import (
	"errors"

	"github.com/google/uuid"
)

var ErrNoOrder = errors.New("a mission has no order")

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
	advanced(progress uint32, fact Fact) uint32
	done(progress uint32) bool
}

// Take Goal tiles for Flag.
type TakeFor struct {
	Flag string
	Goal uint32
}

func (o TakeFor) advanced(progress uint32, fact Fact) uint32 {
	take, ok := fact.(Take)
	return oneMore(progress, o.Goal, ok && take.Flag == o.Flag)
}
func (o TakeFor) done(progress uint32) bool { return progress >= o.Goal }

// Take Goal tiles that Flag held.
type TakeFrom struct {
	Flag string
	Goal uint32
}

func (o TakeFrom) advanced(progress uint32, fact Fact) uint32 {
	take, ok := fact.(Take)
	return oneMore(progress, o.Goal, ok && take.Previous == o.Flag)
}
func (o TakeFrom) done(progress uint32) bool { return progress >= o.Goal }

// Take Goal tiles on Ground, a country's land.
type TakeIn struct {
	Ground string
	Goal   uint32
}

func (o TakeIn) advanced(progress uint32, fact Fact) uint32 {
	take, ok := fact.(Take)
	return oneMore(progress, o.Goal, ok && take.Ground == o.Ground)
}
func (o TakeIn) done(progress uint32) bool { return progress >= o.Goal }

type TakeFromIn struct {
	Flag   string
	Ground string
	Goal   uint32
}

func (o TakeFromIn) advanced(progress uint32, fact Fact) uint32 {
	take, ok := fact.(Take)
	return oneMore(progress, o.Goal, ok && take.Previous == o.Flag && take.Ground == o.Ground)
}
func (o TakeFromIn) done(progress uint32) bool { return progress >= o.Goal }

type CatchBoxes struct {
	Goal uint32
}

func (o CatchBoxes) advanced(progress uint32, fact Fact) uint32 {
	_, caught := fact.(BoxCaught)
	return oneMore(progress, o.Goal, caught)
}
func (o CatchBoxes) done(progress uint32) bool { return progress >= o.Goal }

// Answer Goal quizzes right.
type AnswerQuizzes struct {
	Goal uint32
}

func (o AnswerQuizzes) advanced(progress uint32, fact Fact) uint32 {
	_, right := fact.(RightAnswer)
	return oneMore(progress, o.Goal, right)
}
func (o AnswerQuizzes) done(progress uint32) bool { return progress >= o.Goal }

type BombIn struct {
	Ground string
	Goal   uint32
}

func (o BombIn) advanced(progress uint32, fact Fact) uint32 {
	bomb, ok := fact.(BombLanded)
	return oneMore(progress, o.Goal, ok && bomb.Ground == o.Ground)
}
func (o BombIn) done(progress uint32) bool { return progress >= o.Goal }

// A placement adds one shield: the one that reaches Shields counts the tile.
type ShieldTiles struct {
	Shields uint32
	Goal    uint32
}

func (o ShieldTiles) advanced(progress uint32, fact Fact) uint32 {
	placed, ok := fact.(ShieldPlaced)
	return oneMore(progress, o.Goal, ok && placed.Shields == o.Shields)
}
func (o ShieldTiles) done(progress uint32) bool { return progress >= o.Goal }

type FortifyTiles struct {
	Goal uint32
}

func (o FortifyTiles) advanced(progress uint32, fact Fact) uint32 {
	fortified, ok := fact.(Fortified)
	if !ok || progress >= o.Goal {
		return progress
	}
	return min(progress+fortified.Tiles, o.Goal)
}
func (o FortifyTiles) done(progress uint32) bool { return progress >= o.Goal }

func oneMore(progress, goal uint32, counts bool) uint32 {
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
