package postgres_tile_store_test

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func (s *testSuite) seatsOn(played clicks.MapID) []clicks.Capital {
	var seats []clicks.Capital
	s.Require().NoError(s.store.LoadSeats(context.Background(), played, func(seat clicks.Capital) {
		seats = append(seats, seat)
	}))
	return seats
}

func (s *testSuite) TestAnOpenKeepsTheSeatsWithItsMapOnItsBoard() {
	ctx := context.Background()
	seats := []clicks.Capital{{Country: "gb-eng", Tile: 7}, {Country: "r-0000000001", Tile: 12}}

	_, err := s.store.Open(ctx, battle.On(clicks.TestBoard), owned("gb-eng", 7), seats,
		func(context.Context) (bool, error) { return true, nil })

	s.Require().NoError(err)
	s.Equal(seats, s.seatsOn(battle))
	s.Empty(s.seatsOn(clicks.World))
	boards, err := s.store.Boards(ctx, battle)
	s.Require().NoError(err)
	s.Equal([]clicks.BoardID{clicks.TestBoard}, boards)
}

func (s *testSuite) TestAnOpenKeepRefusesKeepsNoSeat() {
	_, err := s.store.Open(context.Background(), battle.On(clicks.TestBoard), nil, []clicks.Capital{{Country: "gb-eng", Tile: 7}},
		func(context.Context) (bool, error) { return false, nil })

	s.Require().NoError(err)
	s.Empty(s.seatsOn(battle))
}

func (s *testSuite) TestAWipeDeletesTheSeatsOfItsMap() {
	ctx := context.Background()
	_, err := s.store.Open(ctx, battle.On(clicks.TestBoard), nil, []clicks.Capital{{Country: "gb-eng", Tile: 7}},
		func(context.Context) (bool, error) { return true, nil })
	s.Require().NoError(err)

	_, err = s.store.Wipe(ctx, battle, func(context.Context) (bool, error) { return true, nil })

	s.Require().NoError(err)
	s.Empty(s.seatsOn(battle))
}
