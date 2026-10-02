package players

import (
	"crypto/sha256"
	"encoding/hex"
)

const tagLength = 6

// Never leaves the server: it would tell anybody which accounts share an address.
type Tag string

func TagOf(salt string, ip string) Tag {
	sum := sha256.Sum256([]byte(salt + "\x00" + ip))
	return Tag(hex.EncodeToString(sum[:])[:tagLength])
}
