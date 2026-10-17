package cpctx

import "context"

type operatorKey struct{}

func AddOperatorToContext(ctx context.Context, operator string) context.Context {
	return context.WithValue(ctx, operatorKey{}, operator)
}

// Empty on the loopback admin listener: only the operator listener names who calls.
func GetOperator(ctx context.Context) string {
	operator, _ := ctx.Value(operatorKey{}).(string)
	return operator
}
