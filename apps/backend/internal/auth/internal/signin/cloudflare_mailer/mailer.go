package cloudflare_mailer

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
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

const Production = "https://api.cloudflare.com/client/v4"

const maxBodyBytes = 1 << 16

type Config struct {
	AccountID string
	APIToken  string
}

type Sender struct {
	Address string
	Name    string
}

type Mailer struct {
	endpoint string
	token    string
	sender   Sender
	http     *http.Client
}

var _ signin.Mailer = (*Mailer)(nil)

func New(config Config, sender Sender, baseURL string, httpClient *http.Client) (*Mailer, error) {
	if config.AccountID == "" || config.APIToken == "" {
		return nil, errors.New("the cloudflare account id or api token is empty")
	}
	if sender.Address == "" {
		return nil, errors.New("the sender address is empty")
	}
	return &Mailer{
		endpoint: baseURL + "/accounts/" + url.PathEscape(config.AccountID) + "/email/sending/send",
		token:    config.APIToken,
		sender:   sender,
		http:     httpClient,
	}, nil
}

type request struct {
	To      string `json:"to"`
	From    from   `json:"from"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
}

type from struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

type response struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result *struct {
		PermanentBounces []string `json:"permanent_bounces"`
	} `json:"result"`
}

func (m *Mailer) Send(ctx context.Context, to signin.Address, letter signin.Letter) error {
	body, err := json.Marshal(request{
		To: string(to), From: from{Address: m.sender.Address, Name: m.sender.Name},
		Subject: letter.Subject(), Text: letter.Text(), HTML: letter.HTML(),
	})
	if err != nil {
		return fmt.Errorf("failed to encode the letter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := m.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach cloudflare: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	var answer response
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBodyBytes)).Decode(&answer); err != nil {
		return fmt.Errorf("cloudflare answered %d with a body that is not JSON: %w", res.StatusCode, err)
	}
	if res.StatusCode/100 != 2 || !answer.Success {
		reasons := make([]string, 0, len(answer.Errors))
		for _, refusal := range answer.Errors {
			reasons = append(reasons, fmt.Sprintf("%d %s", refusal.Code, refusal.Message))
		}
		return fmt.Errorf("cloudflare answered %d: %s", res.StatusCode, strings.Join(reasons, ", "))
	}
	if answer.Result != nil && slices.Contains(answer.Result.PermanentBounces, string(to)) {
		return errors.New("the address bounced")
	}
	return nil
}
