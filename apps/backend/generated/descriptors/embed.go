// Package descriptors holds what the admin services look like, comments included: a FileDescriptorSet of every
// admin.proto of the monorepo-shared /proto and what it imports, written by `make proto` — this app's copy, the way
// generated/proto is. The connector reads its tools from it (internal/mcp).
//
// The generated Go code registers its descriptors too, but without their comments, and a method's comment is its
// tool's description.
package descriptors

import _ "embed"

//go:embed admin.binpb
var admin []byte

func Admin() []byte {
	return admin
}
