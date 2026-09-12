//go:build testing

package cpconfigs

// FromFile names the file outright. Two test packages need it and no
// production caller does, so it lives behind the tag rather than reading as
// dead code in the binary.
func FromFile(path string) LoadOption {
	return func(p loadParams) loadParams {
		p.path = path
		return p
	}
}
