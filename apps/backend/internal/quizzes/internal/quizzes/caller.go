package quizzes

// Account is the account a click token names: whom a quiz is told of. A caller with none is NoAccount, and asked none.
type Account string

const NoAccount Account = ""

// Caller is who a request comes from, as its click token and its address say. The quizzes read no request: the edge
// that serves one hands its caller over.
type Caller struct {
	Scope   string
	Account Account
	Linked  bool
}
