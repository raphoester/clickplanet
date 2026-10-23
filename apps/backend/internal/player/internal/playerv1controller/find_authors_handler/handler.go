package find_authors_handler

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

const (
	defaultLimit = 20
	maxLimit     = 500
)

var ErrNoName = errors.New("type a part of a name")

type Names interface {
	Find(ctx context.Context, folded string, limit int) ([]players.AccountID, int, error)
}

type Authors interface {
	Authors(ctx context.Context, accounts []players.AccountID) (*playerv1.GetAuthorsResponse, error)
}

func New(names Names, authors Authors) FindAuthorsHandler {
	return FindAuthorsHandler{names: names, authors: authors}
}

type FindAuthorsHandler struct {
	names   Names
	authors Authors
}

func (h FindAuthorsHandler) FindAuthors(
	ctx context.Context,
	req *connect.Request[playerv1.FindAuthorsRequest],
) (*connect.Response[playerv1.FindAuthorsResponse], error) {
	folded := players.FoldedOf(strings.TrimSpace(req.Msg.GetName()))
	if folded == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrNoName)
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultLimit
	}

	accounts, total, err := h.names.Find(ctx, folded, min(limit, maxLimit))
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	authors, err := h.authors.Authors(ctx, accounts)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	res := connect.NewResponse(&playerv1.FindAuthorsResponse{
		Authors: authors.GetAuthors(),
		Total:   uint32(total), //nolint:gosec // an account count, never negative.
	})
	res.Header().Set("Cache-Control", "no-store")
	return res, nil
}
