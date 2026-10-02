package accounts

type IDProvider interface {
	NewID() (AccountID, error)
}

type TokenGenerator interface {
	NewToken() (*Token, error)
}
