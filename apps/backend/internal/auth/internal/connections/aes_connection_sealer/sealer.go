package aes_connection_sealer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
)

// A key per kind, so nothing sealed as one kind opens as another.
const (
	clientInfo  = "clickplanet auth connection client v1"
	requestInfo = "clickplanet auth connection request v1"
	codeInfo    = "clickplanet auth connection code v1"
)

type Sealer struct {
	clients  cipher.AEAD
	requests cipher.AEAD
	codes    cipher.AEAD
}

var _ connections.Sealer = (*Sealer)(nil)

func New(seed []byte) (*Sealer, error) {
	if len(seed) == 0 {
		return nil, errors.New("the seed is empty")
	}

	var aeads [3]cipher.AEAD
	for i, info := range []string{clientInfo, requestInfo, codeInfo} {
		aead, err := aeadOf(seed, info)
		if err != nil {
			return nil, err
		}
		aeads[i] = aead
	}
	return &Sealer{clients: aeads[0], requests: aeads[1], codes: aeads[2]}, nil
}

func aeadOf(seed []byte, info string) (cipher.AEAD, error) {
	key, err := hkdf.Key(sha256.New, seed, nil, info, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive the key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to build the cipher: %w", err)
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("failed to build the AEAD: %w", err)
	}
	return aead, nil
}

// The keys are the format an app holds: a client id registered before a release opens after it. Short, because each
// one travels in a URL.
type sealedClient struct {
	Name         string    `json:"n"`
	RedirectURIs []string  `json:"r"`
	IssuedAt     time.Time `json:"t"`
}

type sealedRequest struct {
	ClientID    string    `json:"c"`
	Client      string    `json:"n"`
	RedirectURI string    `json:"r"`
	Challenge   string    `json:"p"`
	State       string    `json:"s"`
	ExpiresAt   time.Time `json:"e"`
}

type sealedCode struct {
	ID          string             `json:"i"`
	Account     accounts.AccountID `json:"a"`
	ClientID    string             `json:"c"`
	Client      string             `json:"n"`
	RedirectURI string             `json:"r"`
	Challenge   string             `json:"p"`
	ExpiresAt   time.Time          `json:"e"`
}

func (s *Sealer) SealedClient(client *connections.Client) (string, error) {
	snapshot := client.TakeSnapshot()
	return sealed(s.clients, sealedClient(snapshot))
}

func (s *Sealer) OpenedClient(sealedID string) (*connections.Client, error) {
	var opened sealedClient
	if err := open(s.clients, sealedID, &opened); err != nil {
		return nil, fmt.Errorf("%w: %w", connections.ErrInvalidClient, err)
	}
	return connections.ClientSnapshot(opened).Restore() //nolint:wrapcheck // the sentinel is the domain's.
}

func (s *Sealer) SealedRequest(request *connections.Request) (string, error) {
	return sealed(s.requests, sealedRequest(request.TakeSnapshot()))
}

func (s *Sealer) OpenedRequest(sealedRequestValue string) (*connections.Request, error) {
	var opened sealedRequest
	if err := open(s.requests, sealedRequestValue, &opened); err != nil {
		return nil, fmt.Errorf("%w: %w", connections.ErrInvalidRequest, err)
	}
	return connections.RequestSnapshot(opened).Restore() //nolint:wrapcheck // the sentinel is the domain's.
}

func (s *Sealer) SealedCode(code *connections.Code) (string, error) {
	return sealed(s.codes, sealedCode(code.TakeSnapshot()))
}

func (s *Sealer) OpenedCode(sealedCodeValue string) (*connections.Code, error) {
	var opened sealedCode
	if err := open(s.codes, sealedCodeValue, &opened); err != nil {
		return nil, fmt.Errorf("%w: %w", connections.ErrInvalidGrant, err)
	}
	return connections.CodeSnapshot(opened).Restore() //nolint:wrapcheck // the sentinel is the domain's.
}

func sealed(aead cipher.AEAD, value any) (string, error) {
	plain, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to encode: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(aead.Seal(nil, nil, plain, nil)), nil
}

func open(aead cipher.AEAD, value string, into any) error {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("not base64url: %w", err)
	}
	plain, err := aead.Open(nil, nil, raw, nil)
	if err != nil {
		return fmt.Errorf("not sealed by this server: %w", err)
	}
	if err := json.Unmarshal(plain, into); err != nil {
		return fmt.Errorf("sealed and not readable: %w", err)
	}
	return nil
}
