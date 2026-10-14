package get_attachment_rules_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

// The bytes the bucket may still take.
type Room interface {
	Room(ctx context.Context) (int64, error)
}

func New(rules notes.AttachmentRules, room Room) GetAttachmentRulesHandler {
	return GetAttachmentRulesHandler{rules: rules, room: room}
}

type GetAttachmentRulesHandler struct {
	rules notes.AttachmentRules
	room  Room
}

func (h GetAttachmentRulesHandler) GetAttachmentRules(
	ctx context.Context,
	_ *connect.Request[supportv1.GetAttachmentRulesRequest],
) (*connect.Response[supportv1.GetAttachmentRulesResponse], error) {
	room, err := h.room.Room(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read the room the bucket has left: %w", err)
	}
	rules := h.rules.Within(room)

	answer := &supportv1.GetAttachmentRulesResponse{
		MaxPerNote: uint32(max(rules.MaxPerNote, 0)), //nolint:gosec // a count of files.
		MaxBytes:   uint64(max(rules.MaxBytes, 0)),   //nolint:gosec // not negative on this line.
	}
	for _, media := range notes.MediaTypes() {
		answer.ContentTypes = append(answer.ContentTypes, string(media))
	}

	res := connect.NewResponse(answer)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
