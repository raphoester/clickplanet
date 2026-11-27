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

// Who a post pings: the members of one role, everybody in the server, or nobody.
type Audience struct {
	role  RoleID
	quiet bool
}

func RoleAudience(role RoleID) Audience {
	return Audience{role: role}
}

func Everybody() Audience {
	return Audience{}
}

// Nobody is pinged, whatever the text holds: the channel's readers see the post.
func Nobody() Audience {
	return Audience{quiet: true}
}

// The text that pings the audience.
func (a Audience) Mention() string {
	switch {
	case a.quiet:
		return ""
	case a.role == "":
		return "@everyone"
	}
	return "<@&" + string(a.role) + ">"
}

// Quiet is true when the post pings nobody.
func (a Audience) Quiet() bool {
	return a.quiet
}

// No role: the post pings everybody.
func (a Audience) Role() (RoleID, bool) {
	return a.role, a.role != ""
}
