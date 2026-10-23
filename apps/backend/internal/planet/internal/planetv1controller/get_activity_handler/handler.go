package get_activity_handler

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/activity_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in activity_usecase.In) (activity_usecase.Out, error)
}

func New(useCase UseCase, mapTiles uint32) GetActivityHandler {
	return GetActivityHandler{useCase: useCase, mapTiles: mapTiles}
}

type GetActivityHandler struct {
	useCase  UseCase
	mapTiles uint32
}

func (h GetActivityHandler) GetActivity(
	ctx context.Context,
	req *connect.Request[planetv1.GetActivityRequest],
) (*connect.Response[planetv1.GetActivityResponse], error) {
	var in activity_usecase.In
	if span := req.Msg.GetSpan(); span != nil {
		in.Span = span.AsDuration()
	}

	out, err := h.useCase.Execute(ctx, in)
	if errors.Is(err, ledger.ErrInvalidSpan) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err
	}

	slots := make([]*planetv1.ActivitySlot, len(out.Activity.Slots))
	for i, slot := range out.Activity.Slots {
		slots[i] = &planetv1.ActivitySlot{Clicks: count(slot.Clicks), Tiles: count(slot.Tiles), Players: count(slot.Players)}
	}

	return connect.NewResponse(&planetv1.GetActivityResponse{
		Since:   timestamppb.New(out.Activity.Since),
		Until:   timestamppb.New(out.Activity.Until),
		Step:    durationpb.New(out.Activity.Step),
		Slots:   slots,
		Players: count(out.Activity.Players),
		Heatmap: &planetv1.ActivityHeatmap{
			Columns: clicks.GridColumns, Rows: clicks.GridRows, Land: out.Land, Changes: out.Heat,
		},
		Countries: countriesOf(out.Held, out.Activity.Moves),
		Streams:   count(out.Audience.Streams),
		Watchers:  count(out.Audience.Entrants),
		MapTiles:  h.mapTiles,
	}), nil
}

func countriesOf(held map[string]uint32, moves map[string]ledger.Moves) []*planetv1.CountryActivity {
	byCountry := make(map[string]*planetv1.CountryActivity, len(held)+len(moves))
	of := func(country string) *planetv1.CountryActivity {
		if _, ok := byCountry[country]; !ok {
			byCountry[country] = &planetv1.CountryActivity{CountryId: country}
		}
		return byCountry[country]
	}

	for country, tiles := range held {
		of(country).Held = tiles
	}
	for country, moved := range moves {
		row := of(country)
		row.Won, row.Lost = count(moved.Won), count(moved.Lost)
	}

	countries := make([]*planetv1.CountryActivity, 0, len(byCountry))
	for _, row := range byCountry {
		countries = append(countries, row)
	}
	slices.SortFunc(countries, func(a, b *planetv1.CountryActivity) int {
		return cmp.Or(
			cmp.Compare(b.GetHeld(), a.GetHeld()),
			cmp.Compare(b.GetWon(), a.GetWon()),
			cmp.Compare(a.GetCountryId(), b.GetCountryId()),
		)
	})

	return countries
}

func count(n int) uint32 {
	return uint32(n) //nolint:gosec // a count of the ledger's events, bounded by its cap.
}
