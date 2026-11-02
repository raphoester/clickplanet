package discord

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	ErrRefused   = errors.New("discord refused")
	ErrNotMember = errors.New("this user is not in the server")
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

func Mention(role RoleID) string {
	return "<@&" + string(role) + ">"
}
