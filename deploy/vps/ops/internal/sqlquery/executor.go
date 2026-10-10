package sqlquery

import (
	"context"
	"errors"
)

var ErrRefused = errors.New("postgres refused the statement")

type Query struct {
	Statement string
	MaxRows   int
	MaxBytes  int
}

type Result struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	Truncated bool     `json:"truncated"`
}

type Executor interface {
	Execute(ctx context.Context, query Query) (Result, error)
}
