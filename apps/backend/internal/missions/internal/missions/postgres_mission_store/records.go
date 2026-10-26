package postgres_mission_store

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

var ErrUnknownKind = errors.New("not a kind of mission")

// Every mission has an id, a kind and a reward; its kind's own fields are its payload, so a new kind is no migration.
type missionRecord struct {
	ID      uuid.UUID       `json:"id"`
	Kind    string          `json:"kind"`
	Reward  rewardRecord    `json:"reward"`
	Payload json.RawMessage `json:"payload"`
}

type rewardRecord struct {
	Charge string `json:"charge"`
	Amount uint32 `json:"amount"`
}

const (
	kindTakeFor  = "take_for"
	kindTakeFrom = "take_from"
	kindTakeIn   = "take_in"
)

type takeForRecord struct {
	Flag string `json:"flag"`
	Goal uint32 `json:"goal"`
}

type takeFromRecord struct {
	Flag string `json:"flag"`
	Goal uint32 `json:"goal"`
}

type takeInRecord struct {
	Ground string `json:"ground"`
	Goal   uint32 `json:"goal"`
}

func recordOf(mission missions.MissionSnapshot) (missionRecord, error) {
	var (
		kind    string
		payload any
	)
	switch order := mission.Order.(type) {
	case missions.TakeFor:
		kind, payload = kindTakeFor, takeForRecord{Flag: order.Flag, Goal: order.Goal}
	case missions.TakeFrom:
		kind, payload = kindTakeFrom, takeFromRecord{Flag: order.Flag, Goal: order.Goal}
	case missions.TakeIn:
		kind, payload = kindTakeIn, takeInRecord{Ground: order.Ground, Goal: order.Goal}
	default:
		return missionRecord{}, fmt.Errorf("%w: %T", ErrUnknownKind, order)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return missionRecord{}, fmt.Errorf("failed to encode a %s mission: %w", kind, err)
	}
	return missionRecord{
		ID: uuid.UUID(mission.ID), Kind: kind, Payload: encoded,
		Reward: rewardRecord{Charge: string(mission.Reward.Kind), Amount: mission.Reward.Amount},
	}, nil
}

func (r missionRecord) snapshot() (missions.MissionSnapshot, error) {
	order, err := r.order()
	if err != nil {
		return missions.MissionSnapshot{}, err
	}
	return missions.MissionSnapshot{
		ID: missions.MissionID(r.ID), Order: order,
		Reward: missions.Reward{Kind: missions.RewardKind(r.Reward.Charge), Amount: r.Reward.Amount},
	}, nil
}

func (r missionRecord) order() (missions.Order, error) {
	switch r.Kind {
	case kindTakeFor:
		var payload takeForRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.TakeFor{Flag: payload.Flag, Goal: payload.Goal}, decoded(r.Kind, err)
	case kindTakeFrom:
		var payload takeFromRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.TakeFrom{Flag: payload.Flag, Goal: payload.Goal}, decoded(r.Kind, err)
	case kindTakeIn:
		var payload takeInRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.TakeIn{Ground: payload.Ground, Goal: payload.Goal}, decoded(r.Kind, err)
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownKind, r.Kind)
}

func decoded(kind string, err error) error {
	if err != nil {
		return fmt.Errorf("failed to decode a %s mission: %w", kind, err)
	}
	return nil
}
