package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type Mail interface {
	To() string
	Raw() io.ReadCloser
	SetReject(reason string) error
	Forward(to string) error
}

type Backend struct {
	URL      string
	Secret   string
	Operator string
	HTTP     *http.Client
}

// A returned error answers the sending server a temporary failure: it tries again later.
func (b Backend) Deliver(ctx context.Context, mail Mail) error {
	raw := mail.Raw()
	defer func() { _ = raw.Close() }()
	body, err := io.ReadAll(raw)
	if err != nil {
		return fmt.Errorf("failed to read the mail: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+b.Secret)
	req.Header.Set("Content-Type", "message/rfc822")
	req.Header.Set("X-Envelope-To", mail.To())

	res, err := b.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach the backend: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	switch {
	case res.StatusCode == http.StatusRequestEntityTooLarge:
		if err := mail.SetReject("This mail is too large."); err != nil {
			return fmt.Errorf("failed to reject the mail: %w", err)
		}
		return nil
	case res.StatusCode/100 != 2:
		return fmt.Errorf("the backend answered %d", res.StatusCode)
	case res.StatusCode == http.StatusAccepted:
		return nil
	default:
		if err := mail.Forward(b.Operator); err != nil {
			return fmt.Errorf("failed to forward the mail to the operator: %w", err)
		}
		return nil
	}
}
