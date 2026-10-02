//go:build testing

package cpconfigs

func FromFile(path string) LoadOption {
	return func(p loadParams) loadParams {
		p.path = path
		return p
	}
}
