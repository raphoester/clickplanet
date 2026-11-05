package discord

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	ErrRefused   = errors.New("discord refused")
	ErrNotMember = errors.New("this user is not in the server")
	ErrNoRole    = errors.New("the server has no such role")
	ErrInvalidID = errors.New("not a discord id")
)

type (
	UserID    string
	RoleID    string
	ChannelID string
	GuildID   string
)

// Never kept: it joins the player to the server once.
type AccessToken string

// A role's color, 0xRRGGBB. 0 is Discord's "no color", the gray it draws a role in.
type Color int

var snowflake = regexp.MustCompile(`^[0-9]{1,20}$`)

func UserIDOf(id string) (UserID, error) {
	if !snowflake.MatchString(id) {
		return "", fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return UserID(id), nil
}

func ValidID(id string) error {
	if !snowflake.MatchString(id) {
		return fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return nil
}

// Who a post pings: the members of one role, or everybody in the server.
type Audience struct {
	role RoleID
}

func RoleAudience(role RoleID) Audience {
	return Audience{role: role}
}

func Everybody() Audience {
	return Audience{}
}

// The text that pings the audience.
func (a Audience) Mention() string {
	if a.role == "" {
		return "@everyone"
	}
	return "<@&" + string(a.role) + ">"
}

// No role: the post pings everybody.
func (a Audience) Role() (RoleID, bool) {
	return a.role, a.role != ""
}
