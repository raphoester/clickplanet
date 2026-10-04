package accounts

import "crypto/sha256"

type Token struct {
	value string
	hash  TokenHash
}

func TokenOf(value string) *Token {
	sum := sha256.Sum256([]byte(value))
	return &Token{value: value, hash: sum[:]}
}

func (t *Token) Hash() TokenHash {
	return t.hash
}
