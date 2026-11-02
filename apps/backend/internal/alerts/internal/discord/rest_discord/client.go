package rest_discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

type Endpoints struct {
	Authorize string
	Token     string
	API       string
}

var Production = Endpoints{ //nolint:gosec // G101: public endpoint URLs, no credential in them
	Authorize: "https://discord.com/oauth2/authorize",
	Token:     "https://discord.com/api/oauth2/token",
	API:       "https://discord.com/api/v10",
}

// The game's Discord application: its OAuth client, and the bot that lives in the server.
type App struct {
	ClientID     string
	ClientSecret string
	BotToken     string
	RedirectURL  string
	Guild        discord.GuildID
}

const (
	maxBodyBytes = 1 << 16
	userAgent    = "DiscordBot (https://clickplanet.lol, 1)"
	// The scopes a link asks: who the user is, and leave to add them to the server.
	scopes = "identify guilds.join"
	// Discord's code for a user who is not in the server.
	unknownMember = 10007
)

func New(app App, endpoints Endpoints, httpClient *http.Client) *Client {
	return &Client{app: app, endpoints: endpoints, http: httpClient}
}

type Client struct {
	app       App
	endpoints Endpoints
	http      *http.Client
}

func (c *Client) AuthorizationURL(state string, challenge string) string {
	query := url.Values{
		"client_id":             {c.app.ClientID},
		"redirect_uri":          {c.app.RedirectURL},
		"response_type":         {"code"},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return c.endpoints.Authorize + "?" + query.Encode()
}

func (c *Client) Exchange(ctx context.Context, code string, verifier string) (discord.AccessToken, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {c.app.ClientID},
		"client_secret": {c.app.ClientSecret},
		"redirect_uri":  {c.app.RedirectURL},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoints.Token, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to build the token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if _, err := c.status(req, &token); err != nil {
		return "", fmt.Errorf("failed to trade the code: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("%w: the token endpoint answered no access token", discord.ErrRefused)
	}
	return discord.AccessToken(token.AccessToken), nil
}

func (c *Client) UserOf(ctx context.Context, token discord.AccessToken) (discord.UserID, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoints.API+"/users/@me", nil)
	if err != nil {
		return "", fmt.Errorf("failed to build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	var me struct {
		ID string `json:"id"`
	}
	if _, err := c.status(req, &me); err != nil {
		return "", fmt.Errorf("failed to read the user: %w", err)
	}
	user, err := discord.UserIDOf(me.ID)
	if err != nil {
		return "", fmt.Errorf("%w: discord named no user: %w", discord.ErrRefused, err)
	}
	return user, nil
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

func (c *Client) MakeRole(ctx context.Context, name string) (discord.RoleID, error) {
	var made struct {
		ID string `json:"id"`
	}
	if _, err := c.bot(ctx, http.MethodPost, "/guilds/"+string(c.app.Guild)+"/roles",
		map[string]any{"name": name, "mentionable": true}, &made); err != nil {
		return "", fmt.Errorf("failed to make the role %q: %w", name, err)
	}
	if err := discord.ValidID(made.ID); err != nil {
		return "", fmt.Errorf("%w: discord answered no role: %w", discord.ErrRefused, err)
	}
	return discord.RoleID(made.ID), nil
}

// Pings the one role it names, and nobody else whatever the text holds.
func (c *Client) Post(ctx context.Context, channel discord.ChannelID, text string, mention discord.RoleID) error {
	message := map[string]any{
		"content":          text,
		"allowed_mentions": map[string]any{"parse": []string{}, "roles": []discord.RoleID{mention}},
	}
	if _, err := c.bot(ctx, http.MethodPost, "/channels/"+string(channel)+"/messages", message, nil); err != nil {
		return fmt.Errorf("failed to post in the channel %s: %w", channel, err)
	}
	return nil
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
		if reason.Code == unknownMember {
			return res.StatusCode, discord.ErrNotMember
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
