package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// api speaks Connect's JSON encoding over plain HTTP, so this tool needs no generated code.
type api struct {
	base string
	http *http.Client
}

func newAPI(base string) *api {
	return &api{base: base, http: &http.Client{Timeout: 5 * time.Second}}
}

// caller is who a request claims to come from. Caddy sets X-Real-IP in production; here the tool does.
type caller struct {
	ip    string
	token string
}

// call answers the HTTP status, or 0 when no answer came back at all.
func (a *api) call(ctx context.Context, from caller, procedure string, in, out any) (int, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return 0, fmt.Errorf("encode %s: %w", procedure, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.base+procedure, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build %s: %w", procedure, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", from.ip)
	if from.token != "" {
		req.Header.Set("X-Session-Token", from.token)
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", procedure, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("read %s: %w", procedure, err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("%s: HTTP %d: %s", procedure, resp.StatusCode, raw)
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return resp.StatusCode, fmt.Errorf("decode %s: %w", procedure, err)
	}

	return resp.StatusCode, nil
}

func (a *api) createSession(ctx context.Context, ip string) (string, int, error) {
	var out struct {
		Token string `json:"token"`
	}
	status, err := a.call(ctx, caller{ip: ip}, "/session.v1.SessionService/CreateSession",
		map[string]string{"attestationToken": "e2e"}, &out)

	return out.Token, status, err
}

func (a *api) click(ctx context.Context, from caller, tile uint32, country string) (int, error) {
	var out struct{}
	return a.call(ctx, from, "/planet.v1.ClickService/Click",
		map[string]any{"tileId": tile, "countryId": country}, &out)
}

func (a *api) mapDensity(ctx context.Context) (int, error) {
	var out struct{}
	return a.call(ctx, caller{ip: probeIP}, "/planet.v1.ClickService/MapDensity", map[string]any{}, &out)
}

// readMap is every owner from start to end, an unowned tile reading as "".
func (a *api) readMap(ctx context.Context, start, end uint32) (map[uint32]string, error) {
	var out struct {
		StartTileID uint32   `json:"startTileId"`
		Codes       []string `json:"codes"`
		Tiles       string   `json:"tiles"`
	}
	if _, err := a.call(ctx, caller{ip: probeIP}, "/planet.v1.ClickService/GetMap",
		map[string]any{"startTileId": start, "endTileId": end}, &out); err != nil {
		return nil, err
	}

	raw, err := base64.StdEncoding.DecodeString(out.Tiles)
	if err != nil {
		return nil, fmt.Errorf("decode tiles: %w", err)
	}

	owners := make(map[uint32]string, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		code := binary.LittleEndian.Uint16(raw[i:])
		if int(code) >= len(out.Codes) {
			return nil, fmt.Errorf("tile %d references code %d of %d", out.StartTileID+uint32(i/2), code, len(out.Codes))
		}
		owners[out.StartTileID+uint32(i/2)] = out.Codes[code]
	}

	return owners, nil
}

func (a *api) chatHistory(ctx context.Context) ([]string, error) {
	var out struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if _, err := a.call(ctx, caller{ip: probeIP}, "/chat.v1.ChatService/GetHistory", map[string]any{}, &out); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(out.Messages))
	for _, message := range out.Messages {
		ids = append(ids, message.ID)
	}

	return ids, nil
}

// TEST-NET-2: an address no VPN or datacenter list names.
const probeIP = "198.51.100.250"
