package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/raphoester/clickplanet.lol-backend/generated/descriptors"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/mcpcontroller"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/connect_admin_caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/rpc_token_holders"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/usecases/call_tool_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const moduleName = "mcp"

type Config struct {
	Enabled bool

	URL string
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	parsed, err := url.Parse(c.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.Path == "" || parsed.Path == "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("mcp.url %q is not an absolute URL with a path and no query", c.URL)
	}
	return nil
}

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(_ context.Context, props cpbootstrap.Props) error {
			return build(config, props)
		},
	}
}

func build(config Config, props cpbootstrap.Props) error {
	resource, err := url.Parse(config.URL)
	if err != nil {
		return fmt.Errorf("failed to read mcp.url: %w", err)
	}

	catalog, err := tools.CatalogOf(descriptors.Admin(), tools.LeftOut, tools.Hidden)
	if err != nil {
		return fmt.Errorf("failed to read the admin services' tools: %w", err)
	}

	internal, internalURL, err := props.Internal.Dial()
	if err != nil {
		return fmt.Errorf("the connector asks auth who holds a token: %w", err)
	}
	admin, adminURL, err := props.Admin.Dial()
	if err != nil {
		return fmt.Errorf("the connector calls the admin services on their listener: %w", err)
	}

	endpoint := mcpcontroller.NewEndpoint(catalog.Tools(), tools.Instructions, mcpcontroller.MetadataURL(resource),
		rpc_token_holders.New(authv1connect.NewInternalServiceClient(internal, internalURL)),
		call_tool_usecase.New(catalog, connect_admin_caller.New(admin, adminURL)),
		props.Logger,
	)
	metadata := mcpcontroller.NewMetadata(resource)
	inserted, bare := mcpcontroller.MetadataPaths(resource)

	errs := []error{
		props.AdminHTTP.Handle(resource.EscapedPath(), endpoint),
		props.AdminHTTP.Handle("GET "+inserted, metadata),
		props.AdminHTTP.Handle("GET "+bare, metadata),
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("failed to mount the connector: %w", err)
	}

	props.Logger.Info("mcp built", slog.String("url", config.URL), slog.Int("tools", len(catalog.Tools())))
	return nil
}
