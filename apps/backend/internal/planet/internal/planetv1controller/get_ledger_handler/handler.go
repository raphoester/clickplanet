package get_ledger_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/excerpt_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in excerpt_usecase.In) (excerpt_usecase.Out, error)
}

func New(useCase UseCase) GetLedgerHandler {
	return GetLedgerHandler{useCase: useCase}
}

type GetLedgerHandler struct {
	useCase UseCase
}

func (h GetLedgerHandler) GetLedger(
	ctx context.Context,
	req *connect.Request[planetv1.GetLedgerRequest],
) (*connect.Response[planetv1.GetLedgerResponse], error) {
	var in excerpt_usecase.In
	if since := req.Msg.GetSince(); since != nil {
		in.Since = since.AsTime()
	}
	if until := req.Msg.GetUntil(); until != nil {
		in.Until = until.AsTime()
	}

	out, err := h.useCase.Execute(ctx, in)
	if errors.Is(err, excerpt_usecase.ErrInvalidWindow) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(columnsOf(out)), nil
}

func columnsOf(out excerpt_usecase.Out) *planetv1.GetLedgerResponse {
	countries, accounts, scopes := newTable(), newTable(), newTable()
	n := len(out.Lines)
	res := &planetv1.GetLedgerResponse{
		Since:    timestamppb.New(out.Since),
		Until:    timestamppb.New(out.Until),
		At:       make([]uint32, n),
		Tile:     make([]uint32, n),
		Owner:    make([]uint32, n),
		Previous: make([]uint32, n),
		Shields:  make([]uint32, n),
		Clicked:  make([]byte, n),
		Account:  make([]uint32, n),
		Scope:    make([]uint32, n),
	}

	for i, line := range out.Lines {
		res.At[i] = uint32(line.At.Sub(out.Since).Milliseconds()) //nolint:gosec // a window of 72 hours at most.
		res.Tile[i] = line.Update.Tile
		res.Owner[i] = countries.indexOf(line.Update.Value)
		res.Previous[i] = countries.indexOf(line.Update.Previous)
		res.Shields[i] = uint32(line.Update.Shields) //nolint:gosec // a tile holds a few shields.
		if line.Update.Clicked {
			res.Clicked[i] = 1
		}
		res.Account[i] = accounts.indexOf(line.By.Account)
		res.Scope[i] = scopes.indexOf(line.By.Scope)
	}

	res.Countries, res.Accounts, res.Scopes = countries.values, accounts.values, scopes.values
	return res
}

// Index 0 is "", so a zero in a column is no flag, no account or no address.
type table struct {
	values  []string
	indexes map[string]uint32
}

func newTable() *table {
	return &table{values: []string{""}, indexes: map[string]uint32{"": 0}}
}

func (t *table) indexOf(value string) uint32 {
	if index, ok := t.indexes[value]; ok {
		return index
	}

	index := uint32(len(t.values)) //nolint:gosec // as many strings as changes, at most.
	t.indexes[value] = index
	t.values = append(t.values, value)
	return index
}
