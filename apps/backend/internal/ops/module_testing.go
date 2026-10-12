//go:build testing

package ops

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/role"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

var ReaderRole = role.Reader

var errUnknownAssertion = errors.New("nobody signed this assertion")

type KnownAssertions map[string]string

func (k KnownAssertions) Caller(_ context.Context, assertion string) (access.Caller, error) {
	caller, known := k[assertion]
	if !known {
		return "", errUnknownAssertion
	}

	return access.Caller(caller), nil
}

func NewModuleWithKnownAssertions(config Config, assertions KnownAssertions) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error {
			return build(config, props, assertions)
		},
	}
}
