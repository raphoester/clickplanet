package messages

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

const tagLength = 6

var ErrInvalidTag = errors.New("invalid author tag")

// NewTagger builds the stamp from chat.service.tagSalt.
func NewTagger(salt string) Tagger { return Tagger{salt: salt} }

// Tagger stamps the one part of a message its sender cannot forge, which is what makes it a ban's key.
type Tagger struct{ salt string }

// Of hashes the scope and not the address: an IPv6 line renumbers its own host
// part, so a tag over the full address walks out of a ban by doing nothing.
func (t Tagger) Of(ip string) string {
	sum := sha256.Sum256([]byte(t.salt + "\x00" + cpipscope.Of(ip)))
	return hex.EncodeToString(sum[:])[:tagLength]
}

// ParseTag reads a tag an operator typed, as the chat shows it: the '#' is optional and the case is not.
func ParseTag(text string) (string, error) {
	tag := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(text), "#"))

	if len(tag) != tagLength {
		return "", ErrInvalidTag
	}
	if _, err := hex.DecodeString(tag); err != nil {
		return "", ErrInvalidTag
	}

	return tag, nil
}
