package oauthcontroller

import (
	"encoding/json"
	"net/http"
)

const (
	MetadataPath  = "/.well-known/oauth-authorization-server"
	RegisterPath  = "/oauth/register"
	AuthorizePath = "/oauth/authorize"
	TokenPath     = "/oauth/token" //nolint:gosec // a path, not a credential.

	// A registration and a token request are a few hundred bytes.
	MaxBody = 16 << 10
)

func Error(w http.ResponseWriter, status int, code, description string) {
	JSON(w, status, map[string]string{"error": code, "error_description": description})
}

func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
