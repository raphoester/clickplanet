package players

import (
	"crypto/sha256"
	"encoding/hex"
)

const tagLength = 6

// Tag is a salted hash of a caller's address. It never leaves the server: the roster caps the visits of one
// address with it. What the game shows for a guest is its GuestCode, which does not change with the network.
type Tag string

// TagOf is the tag of an address. The same salt and address always give the same tag.
func TagOf(salt string, ip string) Tag {
	sum := sha256.Sum256([]byte(salt + "\x00" + ip))
	return Tag(hex.EncodeToString(sum[:])[:tagLength])
}
