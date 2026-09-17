package signin

// Sealer keeps a flow in a cookie the browser can neither read nor change.
type Sealer interface {
	Sealed(flow *Flow) (string, error)
	// Opened answers ErrFlowInvalid for anything it did not seal.
	Opened(sealed string) (*Flow, error)
}
