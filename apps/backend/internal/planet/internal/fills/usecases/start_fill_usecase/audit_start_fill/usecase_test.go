package audit_start_fill_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/start_fill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/start_fill_usecase/audit_start_fill"
)

type stub struct {
	err error
}

func (s stub) Execute(context.Context, start_fill_usecase.In) error {
	return s.err
}

func TestEveryFillAskedIsLoggedWithItsSourceAndARefusalWithWhy(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&out, nil))
	in := start_fill_usecase.In{Request: "season-0:finale:fill", Source: fills.SourceInternal}

	require.NoError(t, audit_start_fill.New(stub{}, logger).Execute(t.Context(), in))
	refused := errors.New("postgres is down")
	require.ErrorIs(t, audit_start_fill.New(stub{err: refused}, logger).Execute(t.Context(), in), refused)

	logged := out.String()
	assert.Contains(t, logged, `level=WARN msg="charge fill asked" request=season-0:finale:fill source=internal`)
	assert.Contains(t, logged, `msg="charge fill refused" request=season-0:finale:fill source=internal error="postgres is down"`)
}
