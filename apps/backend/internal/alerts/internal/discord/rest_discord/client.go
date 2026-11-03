package rest_discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

type Endpoints struct {
	API string
}

var Production = Endpoints{API: "https://discord.com/api/v10"}

// The game's bot in its server. The bot is the application's that players sign in with: Discord adds a user to a
// server only with an access token of the bot's own application.
type App struct {
	BotToken string
	Guild    discord.GuildID
}

const (
	maxBodyBytes = 1 << 16
	userAgent    = "DiscordBot (https://clickplanet.lol, 1)"
	// Discord's codes for a user who is not in the server, and a role the server does not have.
	unknownMember = 10007
	unknownRole   = 10011
)

func New(app App, endpoints Endpoints, httpClient *http.Client) *Client {
	return &Client{app: app, endpoints: endpoints, http: httpClient}
}

type Client struct {
	app       App
	endpoints Endpoints
	http      *http.Client
}

// A user already in the server keeps its roles: each one is granted then.
func (c *Client) Join(ctx context.Context, user discord.UserID, token discord.AccessToken, roles []discord.RoleID) error {
	status, err := c.bot(ctx, http.MethodPut, c.member(user), map[string]any{"access_token": token, "roles": roles}, nil)
	if err != nil {
		return fmt.Errorf("failed to add the user to the server: %w", err)
	}
	if status == http.StatusCreated {
		return nil
	}
	for _, role := range roles {
		if err := c.Grant(ctx, user, role); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Grant(ctx context.Context, user discord.UserID, role discord.RoleID) error {
	if _, err := c.bot(ctx, http.MethodPut, c.member(user)+"/roles/"+string(role), nil, nil); err != nil {
		return fmt.Errorf("failed to grant the role %s: %w", role, err)
	}
	return nil
}

func (c *Client) Revoke(ctx context.Context, user discord.UserID, role discord.RoleID) error {
	if _, err := c.bot(ctx, http.MethodDelete, c.member(user)+"/roles/"+string(role), nil, nil); err != nil {
		return fmt.Errorf("failed to revoke the role %s: %w", role, err)
	}
	return nil
}

func (c *Client) MakeRole(ctx context.Context, name string, color discord.Color) (discord.RoleID, error) {
	var made struct {
		ID string `json:"id"`
	}
	if _, err := c.bot(ctx, http.MethodPost, c.roles(),
		map[string]any{"name": name, "color": color, "mentionable": true}, &made); err != nil {
		return "", fmt.Errorf("failed to make the role %q: %w", name, err)
	}
	if err := discord.ValidID(made.ID); err != nil {
		return "", fmt.Errorf("%w: discord answered no role: %w", discord.ErrRefused, err)
	}
	return discord.RoleID(made.ID), nil
}

// discord.ErrNoRole when the server has no such role: someone deleted it by hand.
func (c *Client) PaintRole(ctx context.Context, role discord.RoleID, color discord.Color) error {
	if _, err := c.bot(ctx, http.MethodPatch, c.roles()+"/"+string(role), map[string]any{"color": color}, nil); err != nil {
		return fmt.Errorf("failed to paint the role %s: %w", role, err)
	}
	return nil
}

// A role the server no longer has is deleted already.
func (c *Client) DeleteRole(ctx context.Context, role discord.RoleID) error {
	_, err := c.bot(ctx, http.MethodDelete, c.roles()+"/"+string(role), nil, nil)
	if err != nil && !errors.Is(err, discord.ErrNoRole) {
		return fmt.Errorf("failed to delete the role %s: %w", role, err)
	}
	return nil
}

// Pings the audience it names, and nobody else whatever the text holds.
func (c *Client) Post(ctx context.Context, channel discord.ChannelID, text string, audience discord.Audience) error {
	allowed := map[string]any{"parse": []string{"everyone"}}
	if role, ok := audience.Role(); ok {
		allowed = map[string]any{"parse": []string{}, "roles": []discord.RoleID{role}}
	}
	message := map[string]any{"content": text, "allowed_mentions": allowed}
	if _, err := c.bot(ctx, http.MethodPost, "/channels/"+string(channel)+"/messages", message, nil); err != nil {
		return fmt.Errorf("failed to post in the channel %s: %w", channel, err)
	}
	return nil
}

func (c *Client) roles() string {
	return "/guilds/" + string(c.app.Guild) + "/roles"
}

func (c *Client) member(user discord.UserID) string {
	return "/guilds/" + string(c.app.Guild) + "/members/" + string(user)
}

func (c *Client) bot(ctx context.Context, method string, path string, body any, answer any) (int, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("failed to encode the request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoints.API+path, payload)
	if err != nil {
		return 0, fmt.Errorf("failed to build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+c.app.BotToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.status(req, answer)
}

func (c *Client) status(req *http.Request, answer any) (int, error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	res, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("discord is unreachable: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body := io.LimitReader(res.Body, maxBodyBytes)

	switch {
	case res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= http.StatusInternalServerError:
		return res.StatusCode, fmt.Errorf("discord answered %d", res.StatusCode)
	case res.StatusCode >= http.StatusBadRequest:
		var reason struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.NewDecoder(body).Decode(&reason)
		switch reason.Code {
		case unknownMember:
			return res.StatusCode, discord.ErrNotMember
		case unknownRole:
			return res.StatusCode, discord.ErrNoRole
		}
		return res.StatusCode, fmt.Errorf("%w: it answered %d (%d %s%s)", discord.ErrRefused, res.StatusCode,
			reason.Code, reason.Message, reason.Error)
	}

	if answer == nil || res.StatusCode == http.StatusNoContent {
		return res.StatusCode, nil
	}
	if err := json.NewDecoder(body).Decode(answer); err != nil && !errors.Is(err, io.EOF) {
		return res.StatusCode, fmt.Errorf("%w: its answer is not JSON: %w", discord.ErrRefused, err)
	}
	return res.StatusCode, nil
}
