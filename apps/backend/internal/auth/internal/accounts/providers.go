package accounts

// IDProvider gives each new account its id.
type IDProvider interface {
	NewID() (AccountID, error)
}

// TokenGenerator gives each new session its secret.
type TokenGenerator interface {
	NewToken() (*Token, error)
}
