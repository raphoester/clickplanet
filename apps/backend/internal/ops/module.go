package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1/opsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access/cloudflare_assertion_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/access/log_verifier"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler/statement_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops/internal/opsv1controller/query_handler/statement_query/audit_statements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const moduleName = "ops"

// Longer than the role's own statement_timeout, so postgres is the one that says why.
const statementTimeout = 25 * time.Second

func NewModule(config Config) cpbootstrap.Module {
	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(config, props, cloudflare_assertion_verifier.New(ctx, config.Access))
		},
	}
}

func build(config Config, props cpbootstrap.Props, assertions log_verifier.Verifier) error {
	db := cppg.New(config.Database)
	// Opened, not connected: a role that is missing costs a query its answer, never the game its boot.
	if err := db.Open(); err != nil {
		return fmt.Errorf("failed to open the ops module's postgres: %w", err)
	}
	props.Closers.Add("ops-postgres", db.Close)

	service := opsv1controller.OpsService{
		QueryHandler: query_handler.New(audit_statements.New(
			statement_query.NewPostgresQuery(db, statementTimeout), props.Logger,
		)),
	}
	if err := props.RPC.Mount(func(options ...connect.HandlerOption) (string, http.Handler) {
		return opsv1connect.NewOpsServiceHandler(service, options...)
	}, opsv1controller.NewAccessInterceptor(log_verifier.New(assertions, props.Logger))); err != nil {
		return fmt.Errorf("failed to mount ops.v1.OpsService: %w", err)
	}

	props.Logger.Info("ops built", slog.String("user", config.Database.User), slog.String("issuer", config.Access.Issuer))

	return nil
}

type Config struct {
	Enabled  bool
	Access   cloudflare_assertion_verifier.Config
	Database cppg.Config
}

func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	var accessErr, databaseErr error
	if err := c.Access.Validate(); err != nil {
		accessErr = fmt.Errorf("ops.access: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		databaseErr = fmt.Errorf("ops.database: %w", err)
	}

	return errors.Join(accessErr, databaseErr)
}
