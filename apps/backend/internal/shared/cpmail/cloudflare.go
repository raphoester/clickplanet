package cpmail

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
)

const CloudflareAPI = "https://api.cloudflare.com/client/v4"

const maxBodyBytes = 1 << 16

type CloudflareConfig struct {
	AccountID string
	APIToken  string
}

type Cloudflare struct {
	endpoint string
	token    string
	http     *http.Client
}

var _ Mailer = (*Cloudflare)(nil)

func NewCloudflare(config CloudflareConfig, baseURL string, httpClient *http.Client) (*Cloudflare, error) {
	if config.AccountID == "" || config.APIToken == "" {
		return nil, errors.New("the cloudflare account id or api token is empty")
	}
	return &Cloudflare{
		endpoint: baseURL + "/accounts/" + url.PathEscape(config.AccountID) + "/email/sending/send",
		token:    config.APIToken,
		http:     httpClient,
	}, nil
}

type request struct {
	To      string            `json:"to"`
	From    named             `json:"from"`
	ReplyTo string            `json:"reply_to,omitempty"`
	Subject string            `json:"subject"`
	Text    string            `json:"text,omitempty"`
	HTML    string            `json:"html,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type named struct {
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
		MessageID        string   `json:"message_id"`
		PermanentBounces []string `json:"permanent_bounces"`
	} `json:"result"`
}

func (c *Cloudflare) Send(ctx context.Context, mail Mail) (MessageID, error) {
	body, err := json.Marshal(request{
		To: string(mail.To), From: named{Address: string(mail.From.Address), Name: mail.From.Name},
		ReplyTo: string(mail.ReplyTo), Subject: mail.Subject, Text: mail.Text, HTML: mail.HTML, Headers: mail.Headers,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode the mail: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach cloudflare: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	var answer response
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBodyBytes)).Decode(&answer); err != nil {
		return "", fmt.Errorf("cloudflare answered %d with a body that is not JSON: %w", res.StatusCode, err)
	}
	if res.StatusCode/100 != 2 || !answer.Success {
		reasons := make([]string, 0, len(answer.Errors))
		for _, refusal := range answer.Errors {
			reasons = append(reasons, fmt.Sprintf("%d %s", refusal.Code, refusal.Message))
		}
		return "", fmt.Errorf("cloudflare answered %d: %s", res.StatusCode, strings.Join(reasons, ", "))
	}
	if answer.Result == nil {
		return "", errors.New("cloudflare answered no result")
	}
	if slices.Contains(answer.Result.PermanentBounces, string(mail.To)) {
		return "", errors.New("the address bounced")
	}
	id, err := MessageIDOf(answer.Result.MessageID)
	if err != nil {
		return "", fmt.Errorf("cloudflare answered no message id it sent: %w", err)
	}
	return id, nil
}
