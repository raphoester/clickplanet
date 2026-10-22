package connect_admin_caller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
)

const (
	// A reassign, a revert or a reconciliation moves the whole map or every account, paced.
	callTimeout = 5 * time.Minute
	maxAnswer   = 8 << 20
)

func New(client connect.HTTPClient, baseURL string) *Caller {
	return &Caller{client: client, baseURL: baseURL}
}

// A tool's call, made to the admin listener as the app that holds the token: the gate there admits it as that
// operator and writes the call's line, as for the admin page. The arguments are the request in protojson, as is.
type Caller struct {
	client  connect.HTTPClient
	baseURL string
}

func (c *Caller) Call(ctx context.Context, procedure, token string, arguments json.RawMessage) (tools.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	if len(bytes.TrimSpace(arguments)) == 0 || string(arguments) == "null" {
		arguments = json.RawMessage("{}")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+procedure, bytes.NewReader(arguments))
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to build the call: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connect-Protocol-Version", "1")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := c.client.Do(request)
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to call %s: %w", procedure, err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxAnswer))
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to read the answer of %s: %w", procedure, err)
	}
	if response.StatusCode == http.StatusOK {
		return tools.Result{Text: string(body)}, nil
	}

	return refusalOf(procedure, response.StatusCode, body)
}

// Connect's error, as JSON: what Claude reads to try again, a code and a sentence. The error net already took the
// cause off an internal error.
func refusalOf(procedure string, status int, body []byte) (tools.Result, error) {
	var refusal struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &refusal); err != nil || refusal.Code == "" {
		return tools.Result{}, errors.New(procedure + " answered HTTP " + http.StatusText(status))
	}

	text := refusal.Code
	if refusal.Message != "" {
		text += ": " + refusal.Message
	}
	return tools.Result{Text: text, Refused: true}, nil
}
