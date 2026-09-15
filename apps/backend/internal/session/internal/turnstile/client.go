// Package turnstile verifies a Cloudflare Turnstile token against siteverify.
//
// Every failure mode is a refusal: a network error, a non-2xx, a body that is
// not JSON, a token for another action or another site all answer the same way
// a forged token does. Failing open here would make the whole check decorative,
// since an attacker who can reach the backend can also make siteverify
// unreachable from it.
package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	// SiteverifyURL is Cloudflare's endpoint. Never called from the browser:
	// the secret lives here.
	SiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	defaultTimeout = 10 * time.Second

	// Cloudflare documents tokens as up to 2048 characters. Anything longer is
	// refused before it costs a round trip.
	maxTokenLength = 2048

	maxBodyBytes = 1 << 16
)

var ErrRefused = errors.New("attestation refused")

type Config struct {
	Enabled bool

	Secret string

	// The frontend origins siteverify must report. A production value must not
	// contain localhost: one deployment's allowlist is not another's.
	Hostnames []string

	// Must match the data-action the widget was rendered with, so a token
	// minted for some other surface on the same sitekey is not accepted here.
	Action string

	Timeout time.Duration
}

type Client struct {
	secret    string
	hostnames []string
	action    string
	endpoint  string
	http      *http.Client
}

func New(config Config) (*Client, error) {
	if config.Secret == "" {
		return nil, errors.New("turnstile secret is empty")
	}

	if len(config.Hostnames) == 0 {
		return nil, errors.New("turnstile hostnames are empty: every token would be refused")
	}

	if config.Action == "" {
		return nil, errors.New("turnstile action is empty")
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &Client{
		secret:    config.Secret,
		hostnames: slices.Clone(config.Hostnames),
		action:    config.Action,
		endpoint:  SiteverifyURL,
		http:      &http.Client{Timeout: timeout},
	}, nil
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

// Verify answers nil only for a token siteverify accepted, for this action, from
// one of the configured hostnames. Every other outcome wraps ErrRefused, and the
// reason is on the error for the log rather than for the caller.
func (c *Client) Verify(ctx context.Context, token string, remoteIP string) error {
	if token == "" {
		return fmt.Errorf("%w: no token supplied", ErrRefused)
	}

	if len(token) > maxTokenLength {
		return fmt.Errorf("%w: token is %d characters, over the %d limit", ErrRefused, len(token), maxTokenLength)
	}

	form := url.Values{}
	form.Set("secret", c.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: failed to build the siteverify request: %w", ErrRefused, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: siteverify unreachable: %w", ErrRefused, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: siteverify answered %d", ErrRefused, res.StatusCode)
	}

	var body siteverifyResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBodyBytes)).Decode(&body); err != nil {
		return fmt.Errorf("%w: siteverify answered a body that is not JSON: %w", ErrRefused, err)
	}

	if !body.Success {
		return fmt.Errorf("%w: siteverify said no (%s)", ErrRefused, strings.Join(body.ErrorCodes, ", "))
	}

	if body.Action != c.action {
		return fmt.Errorf("%w: token is for action %q, want %q", ErrRefused, body.Action, c.action)
	}

	if !slices.Contains(c.hostnames, body.Hostname) {
		return fmt.Errorf("%w: token is from hostname %q, which is not allowed here", ErrRefused, body.Hostname)
	}

	return nil
}
