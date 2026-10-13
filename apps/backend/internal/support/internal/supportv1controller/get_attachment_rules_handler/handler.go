package get_attachment_rules_handler

import (
	"context"

	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New(rules notes.AttachmentRules) GetAttachmentRulesHandler {
	return GetAttachmentRulesHandler{rules: rules}
}

type GetAttachmentRulesHandler struct {
	rules notes.AttachmentRules
}

func (h GetAttachmentRulesHandler) GetAttachmentRules(
	context.Context,
	*connect.Request[supportv1.GetAttachmentRulesRequest],
) (*connect.Response[supportv1.GetAttachmentRulesResponse], error) {
	answer := &supportv1.GetAttachmentRulesResponse{
		MaxPerNote: uint32(max(h.rules.MaxPerNote, 0)), //nolint:gosec // a count of files.
		MaxBytes:   uint64(max(h.rules.MaxBytes, 0)),   //nolint:gosec // not negative on this line.
	}
	for _, media := range notes.MediaTypes() {
		answer.ContentTypes = append(answer.ContentTypes, string(media))
	}

	res := connect.NewResponse(answer)
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
