package messages

import (
	"crypto/sha256"
	"encoding/hex"
)

const tagLength = 6

// Tag is a salted hash of the sender's IP: a sender cannot forge it.
func Tag(salt string, ip string) string {
	sum := sha256.Sum256([]byte(salt + "\x00" + ip))
	return hex.EncodeToString(sum[:])[:tagLength]
}
