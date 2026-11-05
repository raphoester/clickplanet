//go:build testing

package connections

import (
	"crypto/sha256"
	"encoding/base64"
)

const TestVerifier = "the-verifier-an-app-draws-for-its-request-0001"

func TestChallenge() string {
	sum := sha256.Sum256([]byte(TestVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
