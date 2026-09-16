package signin

// Sealer keeps a flow in a cookie the browser can neither read nor change.
type Sealer interface {
	Seal(flow *Flow) (string, error)
	// Open answers ErrFlowInvalid for anything it did not seal.
	Open(sealed string) (*Flow, error)
}
