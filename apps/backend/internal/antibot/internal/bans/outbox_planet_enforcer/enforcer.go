package outbox_planet_enforcer

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var _ bans.Enforcer = Enforcer{}

func New(outbox cpatomic.Outbox) Enforcer {
	return Enforcer{outbox: outbox}
}

// The antibot speaks planet's words here: a ban is a caller planet refuses, a lifted one a caller it admits.
type Enforcer struct {
	outbox cpatomic.Outbox
}

func (e Enforcer) Enforce(ctx context.Context, caller bans.Caller, until, at time.Time) error {
	if !until.After(at) {
		return e.outbox.Record(ctx, admitted(caller, at)) //nolint:wrapcheck // the banner names the change.
	}
	return e.outbox.Record(ctx, refused(caller, until, at)) //nolint:wrapcheck // the banner names the change.
}

func refused(caller bans.Caller, until, at time.Time) *planetv1.RefuseCaller {
	command := &planetv1.RefuseCaller{Until: timestamppb.New(until), DecidedAt: timestamppb.New(at)}
	if caller.Account != "" {
		command.Caller = &planetv1.RefuseCaller_AccountId{AccountId: caller.Account}
	} else {
		command.Caller = &planetv1.RefuseCaller_Scope{Scope: caller.Scope}
	}
	return command
}

func admitted(caller bans.Caller, at time.Time) *planetv1.AdmitCaller {
	command := &planetv1.AdmitCaller{DecidedAt: timestamppb.New(at)}
	if caller.Account != "" {
		command.Caller = &planetv1.AdmitCaller_AccountId{AccountId: caller.Account}
	} else {
		command.Caller = &planetv1.AdmitCaller_Scope{Scope: caller.Scope}
	}
	return command
}
