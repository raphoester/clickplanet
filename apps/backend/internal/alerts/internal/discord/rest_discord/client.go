package rest_discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

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
	// A 429 that asks to wait this long or less is waited out and sent again, at most twice; a longer one is a failure
	// the caller tries again later.
	maxRetryAfter = 5 * time.Second
	maxRetries    = 2
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

// False for a user who is not in the server, or never was.
func (c *Client) IsMember(ctx context.Context, user discord.UserID) (bool, error) {
	_, err := c.bot(ctx, http.MethodGet, c.member(user), nil, nil)
	switch {
	case errors.Is(err, discord.ErrNotMember):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to ask whether the user is in the server: %w", err)
	}
	return true, nil
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

// The user's nickname in the server; an empty one takes it off, and Discord shows the user's own name again.
// discord.ErrRefused for a user the bot may not rename: the server's owner, or a member whose role is above the bot's.
func (c *Client) Rename(ctx context.Context, user discord.UserID, nick string) error {
	var named any
	if nick != "" {
		named = nick
	}
	if _, err := c.bot(ctx, http.MethodPatch, c.member(user), map[string]any{"nick": named}, nil); err != nil {
		return fmt.Errorf("failed to give the user its nickname: %w", err)
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
	var encoded []byte
	if body != nil {
		var err error
		if encoded, err = json.Marshal(body); err != nil {
			return 0, fmt.Errorf("failed to encode the request: %w", err)
		}
	}
	for retries := 0; ; retries++ {
		req, err := http.NewRequestWithContext(ctx, method, c.endpoints.API+path, payloadOf(encoded))
		if err != nil {
			return 0, fmt.Errorf("failed to build the request: %w", err)
		}
		req.Header.Set("Authorization", "Bot "+c.app.BotToken)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		status, wait, answered := c.status(req, answer)
		if wait <= 0 || retries == maxRetries {
			return status, answered
		}
		if err := sleep(ctx, wait); err != nil {
			return status, fmt.Errorf("%w, and the wait it asked for was cut short: %w", answered, err)
		}
	}
}

func payloadOf(encoded []byte) io.Reader {
	if encoded == nil {
		return nil
	}
	return bytes.NewReader(encoded)
}

func sleep(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the caller says what was cut short.
	case <-timer.C:
		return nil
	}
}

// Discord's Retry-After, in seconds, when it is short enough to wait out; 0 otherwise.
func retryAfterOf(res *http.Response) time.Duration {
	seconds, err := strconv.ParseFloat(res.Header.Get("Retry-After"), 64)
	if err != nil || seconds < 0 {
		return 0
	}
	wait := time.Duration(seconds * float64(time.Second))
	if wait > maxRetryAfter {
		return 0
	}
	// A 0 is a retry now: the smallest wait still says to send again.
	return max(wait, time.Millisecond)
}

// The wait is set for a 429 that Discord asks to send again after a short wait.
func (c *Client) status(req *http.Request, answer any) (int, time.Duration, error) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	res, err := c.http.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("discord is unreachable: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body := io.LimitReader(res.Body, maxBodyBytes)

	switch {
	case res.StatusCode == http.StatusTooManyRequests:
		return res.StatusCode, retryAfterOf(res), fmt.Errorf("discord answered %d", res.StatusCode)
	case res.StatusCode >= http.StatusInternalServerError:
		return res.StatusCode, 0, fmt.Errorf("discord answered %d", res.StatusCode)
	case res.StatusCode >= http.StatusBadRequest:
		var reason struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.NewDecoder(body).Decode(&reason)
		switch reason.Code {
		case unknownMember:
			return res.StatusCode, 0, discord.ErrNotMember
		case unknownRole:
			return res.StatusCode, 0, discord.ErrNoRole
		}
		return res.StatusCode, 0, fmt.Errorf("%w: it answered %d (%d %s%s)", discord.ErrRefused, res.StatusCode,
			reason.Code, reason.Message, reason.Error)
	}

	if answer == nil || res.StatusCode == http.StatusNoContent {
		return res.StatusCode, 0, nil
	}
	if err := json.NewDecoder(body).Decode(answer); err != nil && !errors.Is(err, io.EOF) {
		return res.StatusCode, 0, fmt.Errorf("%w: its answer is not JSON: %w", discord.ErrRefused, err)
	}
	return res.StatusCode, 0, nil
}
