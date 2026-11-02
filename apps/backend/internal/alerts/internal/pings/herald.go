package pings

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/roles"
)

type Roles interface {
	IDOf(ctx context.Context, role roles.Role) (discord.RoleID, error)
}

type Poster interface {
	Post(ctx context.Context, channel discord.ChannelID, text string, audience discord.Audience) error
}

type Channels struct {
	Alerts  discord.ChannelID
	Battles discord.ChannelID
}

func NewHerald(roles Roles, poster Poster, channels Channels, gameURL string) Herald {
	return Herald{roles: roles, poster: poster, channels: channels, gameURL: gameURL}
}

// Says it in the server, to whom it is for.
type Herald struct {
	roles    Roles
	poster   Poster
	channels Channels
	gameURL  string
}

// roles.ErrNoRole when no player of the flag ever linked: nobody to call.
func (h Herald) CallToDefend(ctx context.Context, attack Attack) error {
	role, err := h.roles.IDOf(ctx, roles.Nation(attack.Victim))
	if err != nil {
		return err //nolint:wrapcheck // the book named it.
	}
	audience := discord.RoleAudience(role)
	text := fmt.Sprintf("%s %s %s took %d of your tiles in %s %s in %d minutes. %s",
		audience.Mention(), attack.Attacker.Emoji(), attack.Attacker.Name(), attack.Tiles,
		attack.Ground.Emoji(), attack.Ground.Name(), int(AttackWindow.Minutes()), h.gameURL)
	if err := h.poster.Post(ctx, h.channels.Alerts, text, audience); err != nil {
		return fmt.Errorf("failed to call %s to defend: %w", attack.Victim, err)
	}
	return nil
}

// Everybody in the server: a battle is everyone's, whatever flag they play for.
func (h Herald) CallToBattle(ctx context.Context, start Start) error {
	audience := discord.Everybody()
	text, err := startText(start, audience, h.gameURL)
	if err != nil {
		return err
	}
	if err := h.poster.Post(ctx, h.channels.Battles, text, audience); err != nil {
		return fmt.Errorf("failed to tell the start of %s: %w", start.Key, err)
	}
	return nil
}

// Discord draws <t:…:t> as the time in each reader's own zone.
func startText(start Start, audience discord.Audience, gameURL string) (string, error) {
	until := fmt.Sprintf("<t:%d:t>", start.EndsAt.Unix())
	switch start.Kind {
	case FinaleStart:
		return fmt.Sprintf("%s ⚔️ The Final Battle of season %d is on: %dx points until %s. %s",
			audience.Mention(), start.Season, start.Times, until, gameURL), nil
	case BattleStart:
		return fmt.Sprintf("%s ⚔️ The battle is on: %dx points until %s. %s",
			audience.Mention(), start.Times, until, gameURL), nil
	case SeasonStart:
		return fmt.Sprintf("%s 🌍 Season %d has started on a blank map. %s",
			audience.Mention(), start.Season, gameURL), nil
	}
	return "", fmt.Errorf("%w: %d", ErrUnknownStart, start.Kind)
}
