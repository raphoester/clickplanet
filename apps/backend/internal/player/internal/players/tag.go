package players

import (
	"crypto/sha256"
	"encoding/hex"
)

const tagLength = 6

// Tag is a salted hash of a caller's address. The chat shows it beside every name, so two guests who typed one
// name still look different, and a caller cannot choose it.
type Tag string

// TagOf is the tag of an address. The same salt and address always give the same tag.
func TagOf(salt string, ip string) Tag {
	sum := sha256.Sum256([]byte(salt + "\x00" + ip))
	return Tag(hex.EncodeToString(sum[:])[:tagLength])
}
