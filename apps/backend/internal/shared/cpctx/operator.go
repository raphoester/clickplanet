package cpctx

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Who calls an operator's service, as the admin gate let it in: the operator's account, and the app it connected
// ("Claude"), empty for its own page. A call on the console listener has nobody: no gate stands there.
type Operator struct {
	account cpsession.AccountID
	client  string
}

func OperatorOf(account cpsession.AccountID, client string) Operator {
	return Operator{account: account, client: client}
}

func (o Operator) Account() cpsession.AccountID { return o.account }

func (o Operator) Client() string { return o.client }

type operatorKey struct{}

func AddOperatorToContext(ctx context.Context, operator Operator) context.Context {
	return context.WithValue(ctx, operatorKey{}, operator)
}

func GetOperator(ctx context.Context) Operator {
	operator, _ := ctx.Value(operatorKey{}).(Operator)
	return operator
}
