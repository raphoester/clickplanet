package brevo_audience

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

const Production = "https://api.brevo.com/v3"

const redirectionURL = "https://clickplanet.lol/play"

const maxBodyBytes = 1 << 16

type Config struct {
	APIKey        string
	ListID        int64
	DOITemplateID int64
}

// String leaves out the key: the config is logged at boot.
func (c Config) String() string {
	return fmt.Sprintf("{ListID:%d DOITemplateID:%d}", c.ListID, c.DOITemplateID)
}

type Audience struct {
	baseURL string
	config  Config
	http    *http.Client
}

var _ subscriptions.Audience = (*Audience)(nil)

func New(config Config, baseURL string, httpClient *http.Client) (*Audience, error) {
	if config.APIKey == "" {
		return nil, errors.New("the brevo api key is empty")
	}
	if config.ListID <= 0 || config.DOITemplateID <= 0 {
		return nil, errors.New("the brevo list id or double opt-in template id is not set")
	}
	return &Audience{baseURL: baseURL, config: config, http: httpClient}, nil
}

type contact struct {
	Email            string  `json:"email"`
	ListIDs          []int64 `json:"listIds"`
	UpdateEnabled    bool    `json:"updateEnabled"`
	EmailBlacklisted bool    `json:"emailBlacklisted"`
}

func (a *Audience) Join(ctx context.Context, address subscriptions.Address) error {
	return a.send(ctx, "join", http.MethodPost, "/contacts", contact{
		Email: string(address), ListIDs: []int64{a.config.ListID}, UpdateEnabled: true, EmailBlacklisted: false,
	}, nil)
}

type invitation struct {
	Email          string  `json:"email"`
	IncludeListIDs []int64 `json:"includeListIds"`
	TemplateID     int64   `json:"templateId"`
	RedirectionURL string  `json:"redirectionUrl"`
}

func (a *Audience) Invite(ctx context.Context, address subscriptions.Address) error {
	return a.send(ctx, "invite", http.MethodPost, "/contacts/doubleOptinConfirmation", invitation{
		Email: string(address), IncludeListIDs: []int64{a.config.ListID},
		TemplateID: a.config.DOITemplateID, RedirectionURL: redirectionURL,
	}, nil)
}

type found struct {
	ListIDs          []int64 `json:"listIds"`
	EmailBlacklisted bool    `json:"emailBlacklisted"`
}

func (a *Audience) Joined(ctx context.Context, address subscriptions.Address) (bool, error) {
	var answer found
	err := a.send(ctx, "read", http.MethodGet, contactPath(address), nil, &answer)
	if errors.Is(err, errNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return slices.Contains(answer.ListIDs, a.config.ListID) && !answer.EmailBlacklisted, nil
}

type blacklisting struct {
	EmailBlacklisted bool `json:"emailBlacklisted"`
}

func (a *Audience) Leave(ctx context.Context, address subscriptions.Address) error {
	err := a.send(ctx, "blacklist", http.MethodPut, contactPath(address), blacklisting{EmailBlacklisted: true}, nil)
	if errors.Is(err, errNotFound) {
		return nil
	}
	return err
}

func (a *Audience) Forget(ctx context.Context, address subscriptions.Address) error {
	err := a.send(ctx, "delete", http.MethodDelete, contactPath(address), nil, nil)
	if errors.Is(err, errNotFound) {
		return nil
	}
	return err
}

func contactPath(address subscriptions.Address) string {
	return "/contacts/" + url.QueryEscape(string(address)) + "?identifierType=email_id"
}

var errNotFound = errors.New("brevo has no such contact")

type refusal struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// The operation names the call in errors, which never carry the address.
func (a *Audience) send(ctx context.Context, operation, method, path string, body, answer any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode the brevo request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("failed to build the brevo %s request", operation)
	}
	req.Header.Set("api-key", a.config.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := a.http.Do(req)
	if err != nil {
		var reaching *url.Error
		if errors.As(err, &reaching) {
			err = reaching.Err
		}
		return fmt.Errorf("failed to reach brevo to %s a contact: %w", operation, err)
	}
	defer func() { _ = res.Body.Close() }()

	limited := io.LimitReader(res.Body, maxBodyBytes)
	if res.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if res.StatusCode/100 != 2 {
		var refused refusal
		_ = json.NewDecoder(limited).Decode(&refused)
		return fmt.Errorf("brevo answered %d to %s a contact: %s %s", res.StatusCode, operation, refused.Code, refused.Message)
	}
	if answer == nil {
		return nil
	}
	if err := json.NewDecoder(limited).Decode(answer); err != nil {
		return fmt.Errorf("brevo answered %d to %s a contact with a body that is not JSON: %w", res.StatusCode, operation, err)
	}
	return nil
}
