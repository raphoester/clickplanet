package call_tool_usecase_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/usecases/call_tool_usecase"
)

type catalog struct{}

func (catalog) Tool(name string) (tools.Tool, error) {
	if name != "planet_ban_player" {
		return tools.Tool{}, fmt.Errorf("%w: %q", tools.ErrUnknownTool, name)
	}
	return tools.Tool{Name: name, Procedure: "/planet.v1.AdminService/BanPlayer"}, nil
}

type admin struct {
	called []string
}

func (a *admin) Call(_ context.Context, procedure, token string, arguments json.RawMessage) (tools.Result, error) {
	a.called = append(a.called, procedure+" "+token+" "+string(arguments))
	return tools.Result{Text: "{}"}, nil
}

func TestAToolIsItsProcedureCalledWithTheToken(t *testing.T) {
	called := &admin{}
	useCase := call_tool_usecase.New(catalog{}, called)

	result, err := useCase.Execute(t.Context(), call_tool_usecase.In{Name: "planet_ban_player", Arguments: json.RawMessage(`{"a":1}`), Token: "cpat_ada"})
	require.NoError(t, err)
	assert.Equal(t, tools.Result{Text: "{}"}, result)

	_, err = useCase.Execute(t.Context(), call_tool_usecase.In{Name: "planet_drop_database", Token: "cpat_ada"})
	require.ErrorIs(t, err, tools.ErrUnknownTool)

	assert.Equal(t, []string{`/planet.v1.AdminService/BanPlayer cpat_ada {"a":1}`}, called.called, "nothing is called for a tool nobody offers")
}
