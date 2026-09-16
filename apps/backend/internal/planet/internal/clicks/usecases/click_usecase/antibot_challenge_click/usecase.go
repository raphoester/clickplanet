// Package antibot_challenge_click refuses a caller the guard has asked to prove
// it is a person again, and lets one that has answered straight back in.
//
// It is a second decorator rather than part of antibot_click because the two
// sit on opposite sides of the throttle, and each has to. The shadow ban has to
// sit inside it: a banned caller that stopped being throttled has been told.
// This one has to sit outside it, for the rule the blocklist and the session
// check are outside for — a click refused for its session must not also spend a
// token, or the retry that follows the mint comes back 429 and the web app
// shows the throttle dialog instead of asking the player to prove themselves.
//
// The one click that does pay a token is the one that earns the challenge:
// antibot_click decides that adjacent to the write, where the throttle has
// already charged. That is one token per challenge, and the alternative is
// letting the click that earned it through.
package antibot_challenge_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

// ChallengeGuard is the part of antibot.Guard this decorator uses. Asking is
// also what answers a challenge, so this runs on every click and not only on
// the refused ones.
type ChallengeGuard interface {
	Challenged(scope, session string) bool
}

func New(implementation click_usecase.IUseCase, guard ChallengeGuard) *UseCase {
	return &UseCase{implementation: implementation, guard: guard}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	guard          ChallengeGuard
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	// The same scope the throttle and the ban are keyed on, and the session the
	// edge accepted — which is empty while nothing mints one, and a challenge
	// no caller could answer is never raised.
	scope := cpipscope.Of(cpctx.GetSourceIP(ctx))

	if u.guard.Challenged(scope, cpctx.GetSessionID(ctx)) {
		return click_usecase.Out{}, clicks.ErrChallenged
	}

	return u.implementation.Execute(ctx, in)
}
