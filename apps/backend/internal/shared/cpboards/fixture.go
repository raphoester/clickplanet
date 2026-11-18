//go:build !testing

package cpboards

import "io/fs"

// withFixture is the boards alone: the tests' fixture ships under the testing tag only.
func withFixture(boards fs.FS) fs.FS {
	return boards
}
