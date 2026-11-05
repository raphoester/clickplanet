package postgres_flag_day_store

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
)

var ErrUnknownKind = errors.New("not a kind of flag mission")

// The values of flag_mission_kind; a kind's own fields are its payload.
const (
	kindTakeIn       = "take_in"
	kindBreakShields = "break_shields"
)

type takeInRecord struct {
	Ground string `json:"ground"`
	Goal   uint32 `json:"goal"`
}

type breakShieldsRecord struct {
	Flag string `json:"flag"`
	Goal uint32 `json:"goal"`
}

func recordOf(order flagmissions.Order) (string, []byte, error) {
	var (
		kind    string
		payload any
	)
	switch order := order.(type) {
	case flagmissions.TakeIn:
		kind, payload = kindTakeIn, takeInRecord{Ground: order.Ground, Goal: order.Goal}
	case flagmissions.BreakShields:
		kind, payload = kindBreakShields, breakShieldsRecord{Flag: order.Flag, Goal: order.Goal}
	default:
		return "", nil, fmt.Errorf("%w: %T", ErrUnknownKind, order)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("failed to encode a %s mission: %w", kind, err)
	}
	return kind, encoded, nil
}

func orderOf(kind string, payload []byte) (flagmissions.Order, error) {
	switch kind {
	case kindTakeIn:
		var record takeInRecord
		err := json.Unmarshal(payload, &record)
		return flagmissions.TakeIn{Ground: record.Ground, Goal: record.Goal}, decoded(kind, err)
	case kindBreakShields:
		var record breakShieldsRecord
		err := json.Unmarshal(payload, &record)
		return flagmissions.BreakShields{Flag: record.Flag, Goal: record.Goal}, decoded(kind, err)
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
}

func decoded(kind string, err error) error {
	if err != nil {
		return fmt.Errorf("failed to decode a %s mission: %w", kind, err)
	}
	return nil
}
