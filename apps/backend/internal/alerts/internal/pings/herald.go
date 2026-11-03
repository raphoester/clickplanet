package pings

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
		attack.Ground.Emoji(), attack.Ground.Name(), int(AttackWindow.Minutes()), attack.View.linkFrom(h.gameURL))
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

// The day's top three flags are pinged, each by its role, in #alerts; a flag no player ever linked is named, and
// pinged by nobody.
func (h Herald) TellRanking(ctx context.Context, day EndedDay, ranking Ranking) error {
	var pinged []discord.RoleID
	var text strings.Builder
	fmt.Fprintf(&text, "🏁 **Day %d of season %d is over.** The points it gave:\n", day.Number, day.Season)
	for _, place := range ranking.Won {
		name := place.Flag.Name()
		if place.Rank <= Podium {
			role, err := h.roles.IDOf(ctx, roles.Nation(place.Flag))
			switch {
			case err == nil:
				pinged = append(pinged, role)
				name = role.Mention()
			case !errors.Is(err, roles.ErrNoRole):
				return err //nolint:wrapcheck // the book named it.
			}
		}
		fmt.Fprintf(&text, "%s %s %s +%d\n", badgeOf(place.Rank), place.Flag.Emoji(), name, place.Points)
	}
	text.WriteString("🏆 **The season's ranking:**\n")
	for _, place := range ranking.Table {
		if place.Rank <= TableRows {
			fmt.Fprintf(&text, "%s %s %s %s\n", badgeOf(place.Rank), place.Flag.Emoji(), place.Flag.Name(),
				pointsOf(place.Points))
		}
	}
	text.WriteString(h.gameURL)

	if err := h.poster.Post(ctx, h.channels.Alerts, text.String(), discord.RoleAudience(pinged...)); err != nil {
		return fmt.Errorf("failed to tell the ranking of %s: %w", day.Round, err)
	}
	return nil
}

// A medal for the podium, a keycap down to 10: "4." at a line's start is a Markdown list, which Discord numbers
// again, so two flags tied 2nd would read 2 and 3.
func badgeOf(rank uint32) string {
	switch {
	case rank == 1:
		return "🥇"
	case rank == 2:
		return "🥈"
	case rank == 3:
		return "🥉"
	case rank < 10:
		return fmt.Sprintf("%d\ufe0f\u20e3", rank)
	case rank == 10:
		return "🔟"
	}
	return fmt.Sprintf("#%d", rank)
}

func pointsOf(points uint32) string {
	if points == 1 {
		return "1 point"
	}
	return fmt.Sprintf("%d points", points)
}
