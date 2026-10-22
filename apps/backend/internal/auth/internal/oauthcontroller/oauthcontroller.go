// Package oauthcontroller is the edge of the authorization server an app connects through (connections): plain HTTP,
// since OAuth is forms, redirects and JSON, not Connect. It is served on the admin listener alone, beside the admin
// services, and each handler only maps the wire to its use case.
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

// RFC 6749 5.2's shape, for the token and the registration endpoints alike.
func Error(w http.ResponseWriter, status int, code, description string) {
	JSON(w, status, map[string]string{"error": code, "error_description": description})
}

// Never cached: a token, a client id and a refusal are each one caller's.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
