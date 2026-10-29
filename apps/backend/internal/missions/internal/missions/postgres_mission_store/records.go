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
	kindTakeFor       = "take_for"
	kindTakeFrom      = "take_from"
	kindTakeIn        = "take_in"
	kindTakeFromIn    = "take_from_in"
	kindCatchBoxes    = "catch_boxes"
	kindAnswerQuizzes = "answer_quizzes"
	kindBombIn        = "bomb_in"
	kindShieldTiles   = "shield_tiles"
	kindFortifyTiles  = "fortify_tiles"
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

type takeFromInRecord struct {
	Flag   string `json:"flag"`
	Ground string `json:"ground"`
	Goal   uint32 `json:"goal"`
}

type goalRecord struct {
	Goal uint32 `json:"goal"`
}

type bombInRecord struct {
	Ground string `json:"ground"`
	Goal   uint32 `json:"goal"`
}

type shieldTilesRecord struct {
	Shields uint32 `json:"shields"`
	Goal    uint32 `json:"goal"`
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
	case missions.TakeFromIn:
		kind, payload = kindTakeFromIn, takeFromInRecord{Flag: order.Flag, Ground: order.Ground, Goal: order.Goal}
	case missions.CatchBoxes:
		kind, payload = kindCatchBoxes, goalRecord{Goal: order.Goal}
	case missions.AnswerQuizzes:
		kind, payload = kindAnswerQuizzes, goalRecord{Goal: order.Goal}
	case missions.BombIn:
		kind, payload = kindBombIn, bombInRecord{Ground: order.Ground, Goal: order.Goal}
	case missions.ShieldTiles:
		kind, payload = kindShieldTiles, shieldTilesRecord{Shields: order.Shields, Goal: order.Goal}
	case missions.FortifyTiles:
		kind, payload = kindFortifyTiles, goalRecord{Goal: order.Goal}
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
	case kindTakeFromIn:
		var payload takeFromInRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.TakeFromIn{Flag: payload.Flag, Ground: payload.Ground, Goal: payload.Goal}, decoded(r.Kind, err)
	case kindCatchBoxes:
		var payload goalRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.CatchBoxes{Goal: payload.Goal}, decoded(r.Kind, err)
	case kindAnswerQuizzes:
		var payload goalRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.AnswerQuizzes{Goal: payload.Goal}, decoded(r.Kind, err)
	case kindBombIn:
		var payload bombInRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.BombIn{Ground: payload.Ground, Goal: payload.Goal}, decoded(r.Kind, err)
	case kindShieldTiles:
		var payload shieldTilesRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.ShieldTiles{Shields: payload.Shields, Goal: payload.Goal}, decoded(r.Kind, err)
	case kindFortifyTiles:
		var payload goalRecord
		err := json.Unmarshal(r.Payload, &payload)
		return missions.FortifyTiles{Goal: payload.Goal}, decoded(r.Kind, err)
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownKind, r.Kind)
}

func decoded(kind string, err error) error {
	if err != nil {
		return fmt.Errorf("failed to decode a %s mission: %w", kind, err)
	}
	return nil
}
