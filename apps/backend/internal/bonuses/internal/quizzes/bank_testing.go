//go:build testing

package quizzes

// LoadFixture reads a bank of the test's own, in the format of the shipped one.
func LoadFixture(blob []byte, shares Shares, rated Rated) (*Bank, error) {
	return read(blob, shares, rated)
}
