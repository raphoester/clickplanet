package call_tool_usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
)

type Catalog interface {
	Tool(name string) (tools.Tool, error)
}

type Admin interface {
	Call(ctx context.Context, procedure, token string, arguments json.RawMessage) (tools.Result, error)
}

func New(catalog Catalog, admin Admin) *UseCase {
	return &UseCase{catalog: catalog, admin: admin}
}

type UseCase struct {
	catalog Catalog
	admin   Admin
}

type In struct {
	Name      string
	Arguments json.RawMessage
	Token     string
}

func (u *UseCase) Execute(ctx context.Context, in In) (tools.Result, error) {
	tool, err := u.catalog.Tool(in.Name)
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to find the tool: %w", err)
	}

	result, err := u.admin.Call(ctx, tool.Procedure, in.Token, in.Arguments)
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to call %s: %w", tool.Name, err)
	}
	return result, nil
}
