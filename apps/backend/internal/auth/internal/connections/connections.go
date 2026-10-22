// Package connections is the OAuth 2.1 authorization server an app goes through to reach the admin services: Claude,
// added as a custom connector. The app registers, sends the operator's browser to authorize, the operator lets it in
// on the admin page, and the app trades the code it is sent back with for a pair of tokens (admintokens).
//
// Nothing here is stored. A registered client, a request waiting for the operator and a code are each sealed into
// the string the app carries, under a key derived from auth.secret, so only the tokens are rows.
package connections

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
)

var (
	// The app's client id is not one this server sealed, or it names a redirect it did not register: the browser
	// is not sent back to it, since nobody knows where back is.
	ErrInvalidClient   = errors.New("not a client this server registered")
	ErrInvalidRedirect = errors.New("a redirect URI no app may use")

	ErrInvalidRequest = errors.New("an authorization request this server does not take")
	ErrExpired        = errors.New("past its time")
	ErrInvalidGrant   = errors.New("a code this server does not redeem")
)

const (
	// Long enough for the operator to sign in to the admin page first.
	RequestTTL = 10 * time.Minute
	CodeTTL    = time.Minute

	maxNameRunes = 64
	maxRedirects = 5

	unnamed = "an app"
)

// What an app may be sent back to. Only an app at one of these can be let in, so a page that registers itself
// cannot be handed a code, whoever clicks Allow.
type Redirects []string

func (r Redirects) Allowed(uri string) bool {
	return slices.Contains(r, uri)
}

// A registered app: the name it gave itself and where it may be sent back to.
type Client struct {
	name         string
	redirectURIs []string
	issuedAt     time.Time
}

type ClientSnapshot struct {
	Name         string
	RedirectURIs []string
	IssuedAt     time.Time
}

func NewClient(name string, redirectURIs []string, allowed Redirects, now time.Time) (*Client, error) {
	if len(redirectURIs) == 0 || len(redirectURIs) > maxRedirects {
		return nil, fmt.Errorf("%w: %d redirect URIs, want 1 to %d", ErrInvalidRedirect, len(redirectURIs), maxRedirects)
	}
	for _, uri := range redirectURIs {
		if !allowed.Allowed(uri) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidRedirect, uri)
		}
	}

	return ClientSnapshot{Name: nameOf(name), RedirectURIs: slices.Clone(redirectURIs), IssuedAt: now}.Restore()
}

// A name is shown to the operator and written in the logs: no control character, and a line of it at most.
func nameOf(name string) string {
	cleaned := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, name))
	if runes := []rune(cleaned); len(runes) > maxNameRunes {
		cleaned = strings.TrimSpace(string(runes[:maxNameRunes]))
	}
	if cleaned == "" {
		return unnamed
	}
	return cleaned
}

func (s ClientSnapshot) Restore() (*Client, error) {
	if s.Name == "" || len(s.RedirectURIs) == 0 {
		return nil, fmt.Errorf("%w: a client with no name or no redirect", ErrInvalidClient)
	}
	return &Client{name: s.Name, redirectURIs: slices.Clone(s.RedirectURIs), issuedAt: s.IssuedAt}, nil
}

func (c *Client) TakeSnapshot() ClientSnapshot {
	return ClientSnapshot{Name: c.name, RedirectURIs: slices.Clone(c.redirectURIs), IssuedAt: c.issuedAt}
}

// What a request says it is. The fields are the query of the authorization endpoint.
type Ask struct {
	ResponseType    string
	RedirectURI     string
	Challenge       string
	ChallengeMethod string
	State           string
}

// A request for the operator: the client is sound and so is its redirect, so a refusal from here on is sent back
// to the app, with the state it sent.
func (c *Client) Request(clientID string, ask Ask, now time.Time) (*Request, error) {
	if !slices.Contains(c.redirectURIs, ask.RedirectURI) {
		return nil, fmt.Errorf("%w: %q is not one the client registered", ErrInvalidRedirect, ask.RedirectURI)
	}

	request := RequestSnapshot{
		ClientID: IDOf(clientID), Client: c.name, RedirectURI: ask.RedirectURI, Challenge: ask.Challenge, State: ask.State,
		ExpiresAt: now.Add(RequestTTL),
	}
	switch {
	case ask.ResponseType != "code":
		return nil, &Refusal{URL: request.refusalURL("unsupported_response_type"), cause: ErrInvalidRequest}
	case ask.ChallengeMethod != "S256" || !verifierShaped(ask.Challenge):
		// OAuth 2.1 asks for PKCE on every request, and plain sends the verifier itself.
		return nil, &Refusal{URL: request.refusalURL("invalid_request"), cause: ErrInvalidRequest}
	}

	return request.Restore()
}

// The id a token keeps of its app: a hash, since the client id is a sealed blob as long as a paragraph.
func IDOf(clientID string) string {
	sum := sha256.Sum256([]byte(clientID))
	return hex.EncodeToString(sum[:])
}

// RFC 7636: a challenge is base64url, and a verifier 43 to 128 unreserved characters.
func verifierShaped(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	return !strings.ContainsFunc(value, func(r rune) bool {
		unreserved := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r)
		return !unreserved
	})
}

// A request the app sent the browser back with an OAuth error to.
type Refusal struct {
	URL   string
	cause error
}

func (r *Refusal) Error() string {
	return fmt.Sprintf("%s, sent back to the app", r.cause)
}

func (r *Refusal) Unwrap() error {
	return r.cause
}

// A request waiting for the operator, on the admin page.
type Request struct {
	snapshot RequestSnapshot
}

type RequestSnapshot struct {
	ClientID    string
	Client      string
	RedirectURI string
	Challenge   string
	State       string
	ExpiresAt   time.Time
}

func (s RequestSnapshot) Restore() (*Request, error) {
	if s.ClientID == "" || s.RedirectURI == "" || s.Challenge == "" {
		return nil, fmt.Errorf("%w: a request with no client, redirect or challenge", ErrInvalidRequest)
	}
	return &Request{snapshot: s}, nil
}

func (r *Request) TakeSnapshot() RequestSnapshot {
	return r.snapshot
}

func (r *Request) ExpiryError(now time.Time) error {
	if !now.Before(r.snapshot.ExpiresAt) {
		return fmt.Errorf("%w: the request ended at %s", ErrExpired, r.snapshot.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

func (r *Request) Client() string {
	return r.snapshot.Client
}

// Where the app is: what the operator reads before letting it in, since its name is whatever it chose.
func (r *Request) RedirectHost() string {
	parsed, err := url.Parse(r.snapshot.RedirectURI)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func (r *Request) DeniedURL() string {
	return r.snapshot.refusalURL("access_denied")
}

func (s RequestSnapshot) refusalURL(code string) string {
	return withQuery(s.RedirectURI, url.Values{"error": {code}, "state": {s.State}})
}

// The code the operator lets the app in with, for one minute.
func (r *Request) Approved(account accounts.AccountID, id string, now time.Time) *Code {
	return &Code{snapshot: CodeSnapshot{
		ID: id, Account: account, ClientID: r.snapshot.ClientID, Client: r.snapshot.Client,
		RedirectURI: r.snapshot.RedirectURI, Challenge: r.snapshot.Challenge, ExpiresAt: now.Add(CodeTTL),
	}}
}

// RFC 9207: iss says which server the code is from, so an app talking to two cannot be handed one's code by the other.
func (r *Request) ApprovedURL(code, issuer string) string {
	return withQuery(r.snapshot.RedirectURI, url.Values{"code": {code}, "state": {r.snapshot.State}, "iss": {issuer}})
}

func withQuery(uri string, values url.Values) string {
	if values.Get("state") == "" {
		values.Del("state")
	}
	separator := "?"
	if strings.Contains(uri, "?") {
		separator = "&"
	}
	return uri + separator + values.Encode()
}

type Code struct {
	snapshot CodeSnapshot
}

type CodeSnapshot struct {
	ID          string
	Account     accounts.AccountID
	ClientID    string
	Client      string
	RedirectURI string
	Challenge   string
	ExpiresAt   time.Time
}

func (s CodeSnapshot) Restore() (*Code, error) {
	if s.ID == "" || s.ClientID == "" || s.Challenge == "" {
		return nil, fmt.Errorf("%w: a code with no id, client or challenge", ErrInvalidGrant)
	}
	return &Code{snapshot: s}, nil
}

func (c *Code) TakeSnapshot() CodeSnapshot {
	return c.snapshot
}

func (c *Code) ID() string {
	return c.snapshot.ID
}

func (c *Code) ExpiresAt() time.Time {
	return c.snapshot.ExpiresAt
}

// What the token endpoint is sent with the code: the client it was issued to, the redirect it went to, and the
// verifier whose hash the request carried.
type Redemption struct {
	ClientID    string
	RedirectURI string
	Verifier    string
}

func (c *Code) RedeemError(redemption Redemption, now time.Time) error {
	if !now.Before(c.snapshot.ExpiresAt) {
		return fmt.Errorf("%w: %w", ErrInvalidGrant, ErrExpired)
	}
	if IDOf(redemption.ClientID) != c.snapshot.ClientID || redemption.RedirectURI != c.snapshot.RedirectURI {
		return fmt.Errorf("%w: issued to another client or redirect", ErrInvalidGrant)
	}

	sum := sha256.Sum256([]byte(redemption.Verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if !verifierShaped(redemption.Verifier) || subtle.ConstantTimeCompare([]byte(challenge), []byte(c.snapshot.Challenge)) != 1 {
		return fmt.Errorf("%w: the verifier does not match the challenge", ErrInvalidGrant)
	}

	return nil
}

func (c *Code) Holder() admintokens.Holder {
	return admintokens.Holder{Account: c.snapshot.Account, ClientID: c.snapshot.ClientID, Client: c.snapshot.Client}
}

// Seals what the app carries and opens it again. Each kind has its own key, so a sealed client is never a code.
type Sealer interface {
	SealedClient(client *Client) (string, error)
	OpenedClient(sealed string) (*Client, error)
	SealedRequest(request *Request) (string, error)
	OpenedRequest(sealed string) (*Request, error)
	SealedCode(code *Code) (string, error)
	OpenedCode(sealed string) (*Code, error)
}
