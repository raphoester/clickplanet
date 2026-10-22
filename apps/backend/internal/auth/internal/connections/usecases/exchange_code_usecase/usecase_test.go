package exchange_code_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/aes_connection_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/inmemory_spent_codes"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/exchange_code_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const callback = "https://claude.ai/api/mcp/auth_callback"

var (
	now    = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	ada    = accounts.AccountID{15: 1}
	claude = admintokens.Holder{Account: ada, ClientID: connections.IDOf("the id"), Client: "Claude"}
)

type fixture struct {
	useCase *exchange_code_usecase.UseCase
	tokens  *admintokens.MemoryStore
	code    string
}

func setUp(t *testing.T) fixture {
	t.Helper()
	sealer, err := aes_connection_sealer.New([]byte("the seed"))
	require.NoError(t, err)
	code, err := connections.CodeSnapshot{
		ID: "code 1", Account: ada, ClientID: connections.IDOf("the id"), Client: "Claude", RedirectURI: callback,
		Challenge: connections.TestChallenge(), ExpiresAt: now.Add(connections.CodeTTL),
	}.Restore()
	require.NoError(t, err)
	sealed, err := sealer.SealedCode(code)
	require.NoError(t, err)

	tokens := &admintokens.MemoryStore{}
	return fixture{
		useCase: exchange_code_usecase.New(sealer, inmemory_spent_codes.New(),
			admintokens.NewIssuer(&admintokens.SequentialSecrets{}), tokens, cptime.NewFixedClock(now)),
		tokens: tokens,
		code:   sealed,
	}
}

func in(code, verifier string) exchange_code_usecase.In {
	return exchange_code_usecase.In{
		Code: code, Redemption: connections.Redemption{ClientID: "the id", RedirectURI: callback, Verifier: verifier},
	}
}

func TestACodeBuysAPairForTheOperatorOnce(t *testing.T) {
	f := setUp(t)

	pair, err := f.useCase.Execute(t.Context(), in(f.code, connections.TestVerifier))
	require.NoError(t, err)
	assert.Equal(t, admintokens.Secret("cpat_secret-1"), pair.Access())
	assert.Equal(t, pair.TakeSnapshots(), f.tokens.Kept())
	assert.Equal(t, claude, f.tokens.Kept()[0].Holder)

	_, err = f.useCase.Execute(t.Context(), in(f.code, connections.TestVerifier))
	require.ErrorIs(t, err, connections.ErrInvalidGrant, "spent")
	assert.Len(t, f.tokens.Kept(), 2)
}

func TestAWrongVerifierSpendsNothing(t *testing.T) {
	f := setUp(t)

	_, err := f.useCase.Execute(t.Context(), in(f.code, connections.TestVerifier+"x"))
	require.ErrorIs(t, err, connections.ErrInvalidGrant)
	assert.Empty(t, f.tokens.Kept())

	_, err = f.useCase.Execute(t.Context(), in(f.code, connections.TestVerifier))
	require.NoError(t, err, "the code is still good for its own app")
}

func TestACodeThisServerDidNotSealBuysNothing(t *testing.T) {
	f := setUp(t)

	_, err := f.useCase.Execute(t.Context(), in("made up", connections.TestVerifier))

	require.ErrorIs(t, err, connections.ErrInvalidGrant)
}
