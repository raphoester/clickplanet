package discord

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
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

// Who a post pings: everybody in the server, or the members of some roles. No role pings nobody.
type Audience struct {
	everybody bool
	roles     []RoleID
}

func RoleAudience(roles ...RoleID) Audience {
	return Audience{roles: roles}
}

func Everybody() Audience {
	return Audience{everybody: true}
}

// The text that pings the audience.
func (a Audience) Mention() string {
	if a.everybody {
		return "@everyone"
	}
	mentions := make([]string, 0, len(a.roles))
	for _, role := range a.roles {
		mentions = append(mentions, role.Mention())
	}
	return strings.Join(mentions, " ")
}

func (a Audience) Everybody() bool {
	return a.everybody
}

// Never nil: Discord reads an empty list as no role, and null as none given.
func (a Audience) Roles() []RoleID {
	return append([]RoleID{}, a.roles...)
}

// The text that pings the role's members, and that Discord draws as its name.
func (r RoleID) Mention() string {
	return "<@&" + string(r) + ">"
}
