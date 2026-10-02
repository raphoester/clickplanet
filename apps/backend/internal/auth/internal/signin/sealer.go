package signin

type Sealer interface {
	Sealed(flow *Flow) (string, error)
	Opened(sealed string) (*Flow, error)
}
