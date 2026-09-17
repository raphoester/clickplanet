package signin

// Config is sign-in with a provider. Off, StartSignIn and CompleteSignIn answer Unimplemented.
type Config struct {
	Enabled bool

	// The frontend page every provider sends the browser back to, exactly as registered with each provider.
	RedirectURL string
}
