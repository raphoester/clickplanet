package standings

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

var ErrNoLine = errors.New("the account took no tile this season")

type Store interface {
	RecordTake(ctx context.Context, season calendar.Number, take Take) error
	Line(ctx context.Context, season calendar.Number, account AccountID) (Line, error)
	Lines(ctx context.Context, season calendar.Number, country Country, from Cursor, limit int) ([]Line, error)
	DeleteAccount(ctx context.Context, account AccountID) error
}

type Lines interface {
	Line(ctx context.Context, season calendar.Number, account AccountID) (Line, error)
	Lines(ctx context.Context, season calendar.Number, country Country, from Cursor, limit int) ([]Line, error)
}

type Players interface {
	Players(ctx context.Context, accounts []AccountID) (map[AccountID]Player, error)
}

type Color int32

type Player struct {
	Name  string
	Color Color
	Guest bool
}

func (p Player) Ranked() bool {
	return !p.Guest
}

type Standing struct {
	Rank   uint32
	Line   Line
	Player Player
}

type Place struct {
	Line        Line
	GlobalRank  uint32
	CountryRank uint32
}

const (
	Shown = 10
	page  = 200
)

type Board struct {
	lines   Lines
	players Players
}

func NewBoard(lines Lines, players Players) Board {
	return Board{lines: lines, players: players}
}

func (b Board) Top(ctx context.Context, season calendar.Number, country Country) ([]Standing, error) {
	top := make([]Standing, 0, Shown)
	from := Start
	for {
		lines, err := b.lines.Lines(ctx, season, country, from, page)
		if err != nil {
			return nil, fmt.Errorf("failed to read the standings: %w", err)
		}
		ranked, err := b.ranked(ctx, lines)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			player, ok := ranked[line.Account]
			if !ok {
				continue
			}
			top = append(top, Standing{Rank: rankAfter(top, line), Line: line, Player: player})
			if len(top) == Shown {
				return top, nil
			}
		}
		if len(lines) < page {
			return top, nil
		}
		from = lines[len(lines)-1].Cursor()
	}
}

func (b Board) Place(ctx context.Context, season calendar.Number, account AccountID) (Place, error) {
	line, err := b.lines.Line(ctx, season, account)
	if errors.Is(err, ErrNoLine) {
		return Place{}, nil
	}
	if err != nil {
		return Place{}, fmt.Errorf("failed to read the account's standing: %w", err)
	}
	ranked, err := b.ranked(ctx, []Line{line})
	if err != nil {
		return Place{}, err
	}
	if _, ok := ranked[account]; !ok {
		return Place{Line: line}, nil
	}

	place := Place{Line: line, GlobalRank: 1, CountryRank: 1}
	from := Start
	for {
		lines, err := b.lines.Lines(ctx, season, "", from, page)
		if err != nil {
			return Place{}, fmt.Errorf("failed to read the standings: %w", err)
		}
		above := lines
		if end := slices.IndexFunc(lines, func(other Line) bool { return other.Tiles <= line.Tiles }); end >= 0 {
			above = lines[:end]
		}
		ranked, err := b.ranked(ctx, above)
		if err != nil {
			return Place{}, err
		}
		for _, other := range above {
			if _, ok := ranked[other.Account]; !ok {
				continue
			}
			place.GlobalRank++
			if other.Country == line.Country {
				place.CountryRank++
			}
		}
		if len(above) < page {
			return place, nil
		}
		from = above[len(above)-1].Cursor()
	}
}

func (b Board) ranked(ctx context.Context, lines []Line) (map[AccountID]Player, error) {
	if len(lines) == 0 {
		return map[AccountID]Player{}, nil
	}
	players, err := b.players.Players(ctx, accountsOf(lines))
	if err != nil {
		return nil, fmt.Errorf("failed to ask who the players are: %w", err)
	}
	maps.DeleteFunc(players, func(_ AccountID, player Player) bool { return !player.Ranked() })
	return players, nil
}

func rankAfter(top []Standing, line Line) uint32 {
	if len(top) > 0 && top[len(top)-1].Line.Tiles == line.Tiles {
		return top[len(top)-1].Rank
	}
	return uint32(len(top) + 1) //nolint:gosec // at most Shown.
}

func accountsOf(lines []Line) []AccountID {
	accounts := make([]AccountID, len(lines))
	for i, line := range lines {
		accounts[i] = line.Account
	}
	return accounts
}
