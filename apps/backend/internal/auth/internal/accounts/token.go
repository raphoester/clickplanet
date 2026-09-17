package accounts

import "crypto/sha256"

// Token is a session's secret: the value goes in the cookie, the hash in the table.
type Token struct {
	Value string
	Hash  TokenHash
}

func TokenOf(value string) *Token {
	sum := sha256.Sum256([]byte(value))
	return &Token{Value: value, Hash: sum[:]}
}
