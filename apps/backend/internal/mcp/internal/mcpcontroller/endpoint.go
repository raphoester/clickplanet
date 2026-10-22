// Package mcpcontroller is the connector's edge: the Model Context Protocol over Streamable HTTP, answered with plain
// JSON (the server never streams), and the metadata that points an app at the authorization server.
package mcpcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/usecases/call_tool_usecase"
)

const (
	latestVersion = "2025-06-18"
	maxRequest    = 1 << 20

	// JSON-RPC 2.0's codes.
	parseError     = -32700
	invalidRequest = -32600
	methodNotFound = -32601
	invalidParams  = -32602
)

// The versions this server speaks, newest first: a client asking for another is answered the newest.
var versions = []string{latestVersion, "2025-03-26", "2024-11-05"}

type Holders interface {
	HolderError(ctx context.Context, token string) error
}

type Calls interface {
	Execute(ctx context.Context, in call_tool_usecase.In) (tools.Result, error)
}

type Endpoint struct {
	tools        []Tool
	instructions string
	metadataURL  string
	holders      Holders
	calls        Calls
	logger       *slog.Logger
}

// metadataURL is where the 401 points an app that holds no token: the protected resource metadata.
func NewEndpoint(catalog []tools.Tool, instructions, metadataURL string, holders Holders, calls Calls, logger *slog.Logger) *Endpoint {
	listed := make([]Tool, 0, len(catalog))
	for _, tool := range catalog {
		listed = append(listed, toolOf(tool))
	}
	return &Endpoint{
		tools: listed, instructions: instructions, metadataURL: metadataURL, holders: holders, calls: calls, logger: logger,
	}
}

// The wire's shape of a tool. Every call needs Claude's user to have connected as the operator; the hints tell it
// which calls to make without asking (claude.ai lets the user set each tool's permission besides).
type Tool struct {
	Name        string       `json:"name"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	InputSchema tools.Schema `json:"inputSchema"`
	Annotations annotations  `json:"annotations"`
}

type annotations struct {
	Title           string `json:"title"`
	ReadOnlyHint    bool   `json:"readOnlyHint"`
	DestructiveHint bool   `json:"destructiveHint"`
	IdempotentHint  bool   `json:"idempotentHint"`
	OpenWorldHint   bool   `json:"openWorldHint"`
}

func toolOf(tool tools.Tool) Tool {
	return Tool{
		Name: tool.Name, Title: tool.Title, Description: tool.Description, InputSchema: tool.InputSchema,
		Annotations: annotations{
			Title: tool.Title, ReadOnlyHint: tool.ReadOnly, DestructiveHint: !tool.ReadOnly, IdempotentHint: tool.ReadOnly,
		},
	}
}

type message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// A request has an id; a notification, and a response the client sends back, need no answer.
func (m message) request() bool {
	return len(m.ID) > 0 && m.Method != ""
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// No stream from the server, and no session to end.
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "POST a JSON-RPC message", http.StatusMethodNotAllowed)
		return
	}

	token, ok := e.admitted(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequest))
	if err != nil {
		e.write(w, response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: parseError, Message: "unreadable body"}})
		return
	}

	// The 2025-03-26 revision sends a batch as an array; the ones after it send one message a POST.
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var batch []message
		if err := json.Unmarshal(trimmed, &batch); err != nil || len(batch) == 0 {
			e.write(w, response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: parseError, Message: "not a JSON-RPC batch"}})
			return
		}
		var answers []response
		for _, asked := range batch {
			if asked.request() {
				answers = append(answers, e.answer(r.Context(), token, asked))
			}
		}
		e.writeAll(w, answers)
		return
	}

	var asked message
	if err := json.Unmarshal(trimmed, &asked); err != nil {
		e.write(w, response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{Code: parseError, Message: "not a JSON-RPC message"}})
		return
	}
	if !asked.request() {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	e.write(w, e.answer(r.Context(), token, asked))
}

// RFC 9728 and the MCP spec: no token, or a token nobody holds, is a 401 that says where to get one.
func (e *Endpoint) admitted(w http.ResponseWriter, r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata=%q`, e.metadataURL))
		http.Error(w, "connect to ClickPlanet first", http.StatusUnauthorized)
		return "", false
	}

	err := e.holders.HolderError(r.Context(), token)
	switch {
	case err == nil:
		return token, true
	case errors.Is(err, tools.ErrNoToken):
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer error="invalid_token", resource_metadata=%q`, e.metadataURL))
		http.Error(w, "the token lapsed: connect again", http.StatusUnauthorized)
	case errors.Is(err, tools.ErrNotAnOperator):
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer error="insufficient_scope", resource_metadata=%q`, e.metadataURL))
		http.Error(w, "the account that connected no longer operates", http.StatusForbidden)
	default:
		e.logger.Error("failed to check an MCP token", slog.Any("error", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
	return "", false
}

func (e *Endpoint) answer(ctx context.Context, token string, asked message) response {
	answered := response{JSONRPC: "2.0", ID: asked.ID}
	if asked.JSONRPC != "2.0" {
		answered.Error = &rpcError{Code: invalidRequest, Message: "jsonrpc must be 2.0"}
		return answered
	}

	switch asked.Method {
	case "initialize":
		answered.Result = e.initialized(asked.Params)
	case "ping":
		answered.Result = struct{}{}
	case "tools/list":
		answered.Result = map[string]any{"tools": e.tools}
	case "tools/call":
		answered.Result, answered.Error = e.called(ctx, token, asked.Params)
	default:
		answered.Error = &rpcError{Code: methodNotFound, Message: "no method " + asked.Method}
	}
	return answered
}

func (e *Endpoint) initialized(params json.RawMessage) map[string]any {
	var asked struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &asked)

	version := latestVersion
	if slices.Contains(versions, asked.ProtocolVersion) {
		version = asked.ProtocolVersion
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": "clickplanet", "title": "ClickPlanet", "version": "1"},
		"instructions":    e.instructions,
	}
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// A refusal of the service is the tool's result, so Claude reads it and tries again; only a tool nobody offers is a
// protocol error.
func (e *Endpoint) called(ctx context.Context, token string, params json.RawMessage) (any, *rpcError) {
	var asked struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &asked); err != nil || asked.Name == "" {
		return nil, &rpcError{Code: invalidParams, Message: "tools/call needs a name"}
	}

	result, err := e.calls.Execute(ctx, call_tool_usecase.In{Name: asked.Name, Arguments: asked.Arguments, Token: token})
	switch {
	case errors.Is(err, tools.ErrUnknownTool):
		return nil, &rpcError{Code: invalidParams, Message: "no tool " + asked.Name}
	case err != nil:
		e.logger.Error("failed to call a tool", slog.String("tool", asked.Name), slog.Any("error", err))
		result = tools.Result{Text: "internal error: the call did not reach the service, or its answer was lost", Refused: true}
	}

	return map[string]any{"content": []content{{Type: "text", Text: result.Text}}, "isError": result.Refused}, nil
}

func (e *Endpoint) write(w http.ResponseWriter, answered response) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(answered)
}

func (e *Endpoint) writeAll(w http.ResponseWriter, answers []response) {
	if len(answers) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(answers)
}
