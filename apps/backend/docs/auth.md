# Auth (`internal/auth/`)

**One module admits a caller: Turnstile, then the account its cookie holds, then the click token.** Every clicking browser gets an account, a guest one until it signs in; signing in with Google, Discord or an emailed code links to the same row, so a guest's history needs no merge. Off by default (`auth.enabled`); off, `auth.v1` and `session.v1` 404.

It was two modules, `session` and `auth`, for one PR. The mint was auth's only caller, and "not a bot" and "who" are decided on the same request for the same purpose, so the line between them cut through one piece of work.

```
internal/auth/internal/
  accounts/                                AccountID, TokenHash, Account, Identity, Session, Token, Lifetime, OutcomeOf, the cookie; the Store, IDProvider and TokenGenerator ports
    postgres_account_store/                accounts, identities and sessions in the auth schema
    inmemory_account_store/                the same port in maps, behind the testing tag
    uuid_id_provider/  random_token_generator/
    usecases/create_session_usecase/       attest, resume or start a guest, mint
    usecases/create_anonymous_session_usecase/   deprecated: attest, mint with no account
    usecases/sign_out_usecase/  sign_out_everywhere_usecase/  delete_account_usecase/
    usecases/prune_guests_usecase/         deletes idle guests: Executor, Runner, and log_prune_guests
  signin/                                  Flow (state, PKCE verifier, nonce, intent), Provider, Providers, Offer, Sealer, the cp_oauth cookie;
                                           Admitter (signs a proven identity in, for both paths); LinkTarget;
                                           Address, Challenge, Challenges, Post, Letter, the cp_email cookie
    google_identity_provider/  discord_identity_provider/  oauth_http/
    aes_flow_sealer/  random_secret_generator/  random_code_generator/
    cloudflare_mailer/  log_mailer/  embedded_disposable_domains/
    usecases/start_sign_in_usecase/  complete_sign_in_usecase/
    usecases/start_email_sign_in_usecase/  complete_email_sign_in_usecase/
  attestation/                             Attester, ErrAttestationFailed
    turnstile/  turnstile_attester/  open_attester/
  authv1controller/                        AuthService, InternalService and AdminService: one handler package per procedure, authprovider for the enum
    get_me_handler/me_query/               PostgresQuery: GetMeResponse from the cookie, straight from SQL
    get_account_handler/account_query/     PostgresQuery: GetAccountResponse, for another module
    get_accounts_handler/accounts_query/   PostgresQuery: GetAccountsResponse, a page at once
    get_operator_handler/operator_query/   PostgresQuery: GetOperatorResponse, the cp_admin cookie's live session if it operates
  adminsignin/                             the admin page's sign-in: Candidate, Session (Snapshot), Issuer, the cookies, the Store port
    postgres_admin_session_store/          the Store over identities, accounts and admin_sessions
    usecases/start_admin_sign_in_usecase/  complete_admin_sign_in_usecase/  sign_out_admin_usecase/
  admingate/                               what the admin listener sits behind; NewAdminGate builds it, main injects it
    admission/                             Page (the admin page's origin), Visit, Operator, the refusals
      rpc_operators/                       who operates, from InternalService/GetOperator
      usecases/admit_usecase/              the origin, then who calls — and log_admit/, which logs each refusal
    guard/                                 the interceptor: a refusal's code, the operator on the context
      log_calls/                           one line per admitted call
  sessionv1controller/                     deprecated SessionService
    operator_query/                        PostgresQuery: the cookie's live session, its account and whether it operates
    log_refusals/                          logs each caller it refused
  migrations/
```

- **`session.v1.SessionService/CreateSession` is deprecated**, in the proto and in the code. It mints a token with no account for clients that predate accounts, and goes once none calls it. Both paths spend one mint budget, so the old one is not a second allowance.
- **The cookie is `cp_sid`**: a 32-byte token from `random_token_generator`, `HttpOnly; Secure; SameSite=Lax; Path=/`, host-only on the API's domain. The API and the frontend are the same site, so it is not a third-party cookie. **Only its SHA-256 is stored** (`sessions.token_hash`), so a copy of the table signs nobody in. Caddy redacts `Cookie` and `Set-Cookie` in its logs.
- **The rules are on `accounts.Session`**: `GuestSession` builds one with a full `guestTTL` (90 days) and `LinkedSession` with a full `linkedTTL` (30 days), `ExpiryError` is `ErrSessionExpired` past it, `Extendable(now, lifetime)` is whether `extendEvery` (24h) has passed since the last extension, `Extended(now, lifetime)` is a copy whose expiry moved by the TTL of its kind, and `Cookie` is its `Set-Cookie`. `Session.Linked` is read by the store (does the account have an identity), never written, so every session of an account extends as a linked one once any browser links it. `create_session_usecase` only orders them: attest, read the cookie, find the session, check it, extend and save it when due — or, when there is no cookie, no such session or an expired one, start and store a guest — then mint.
- **Only a caller that passed attestation gets an account**, so bots that fail Turnstile make no rows. The mint throttle bounds how many guests one address makes.
- **A failing database fails the mint.** No fallback to a token with no account: the error net answers `internal`.
- **Absence is a sentinel, never `nil, nil`**: `ErrNoSessionCookie`, `ErrSessionNotFound` (the port's, for an unknown token hash), `ErrSessionExpired`, and `ErrNoAccount`. `GetMe` answers `me_query.ErrNoAccount` as `Unauthenticated`, creates, extends and saves nothing, and answers `no-store`.
- **Ids and tokens are injected** (`IDProvider`, `TokenGenerator`), like the clock. Tests use `accounts.SequentialIDs` and `SequentialTokens` (behind the tag), so they assert exact ids.
- **`accounts.StoreContractSuite` is the port's behaviour**, like `clicks.TileStorageContractSuite`. Both stores embed it: postgres adds only what the port cannot show (the token is never stored, `last_seen_at`, an unverified email is `NULL`), and the use cases are tested over the in-memory one, which can `FailWith` an error.
- **No cache.** Mints are one per 30s per address, so one indexed read each is cheap, and a sign-out has nothing to invalidate.
- **`InternalService/GetAccount(account_id)`** answers `linked`: whether the account signed in with a provider, read by `account_query` (an identity exists). An unknown account, or an id that is not one (`accounts.AccountIDOf`), is `linked` false and not an error; a store failure is. `player` asks it before it gives an account a username.
- **`InternalService/GetAccounts(account_ids)`** is a page of accounts at once, each with `linked` and `created_at_unix_ms`, in one statement (`accounts_query`: `id = ANY(...)`, in id order, linked when an identity exists), for the title reconciliation. An account it does not know is left out; an id that is not one is `InvalidArgument`, since a caller holding one has a bug. It is `NO_SIDE_EFFECTS`.
- **`create_session_usecase` mints whether the account is linked** (`Session.Linked`, read by the store), so the token says it and `planet` gives a linked account its faster bucket.
- **`planet` reads the account off the token**: the session interceptor puts it on the context (`cpctx.GetAccount`, and `cpctx.GetLinked`), and the throttle, the bans and the ledger key on it beside the scope — see [Two buckets per click](rate-limiting.md#two-buckets-per-click), [Anti-bot](antibot.md) and [Manual bans](admin.md#manual-bans-findplayers-topplayers-banplayer-unbanplayer-revertplayer-inspectplayer).

## Signing in (`internal/auth/internal/signin/`)

**OAuth 2.0 authorization code with PKCE, run by this server, over two RPCs.** No raw HTTP route and no Caddy change: the provider sends the browser to the frontend's callback page, and that page calls the API.

- **`StartSignIn(provider, intent)`** draws a `signin.Flow` — state, PKCE verifier and nonce, 32 random bytes each, and the intent — seals it into the `cp_oauth` cookie (10 minutes, same attributes as `cp_sid`) and answers the provider's authorization URL. Nothing is stored on the server.
- **The intent is sign in or link** (`accounts.Intent`). Unset on the wire is sign in, and so is a flow sealed before intents existed: `IntentSignIn` is the zero value. The menu's "Sign in with" buttons send sign in, and "Link" sends link.
- **A link needs an account when it starts.** With no live `cp_sid`, `StartSignIn` answers `Unauthenticated` and sets no cookie. The flow keeps the account id, and `CompleteSignIn` refuses a link whose browser is on another account or none by then (`Flow.AccountError`, `ErrFlowInvalid`), before the provider is asked.
- **`CompleteSignIn(code, state)`** opens the cookie, checks the state (constant time) and the expiry, trades the code with the verifier, and clears `cp_oauth` on every answer it maps, success or refusal. A browser that did not start the sign-in has no cookie that matches, which is the whole CSRF defence: a code carried to a victim's browser is refused before the provider is asked.
- **The cookie is AES-256-GCM** (`aes_flow_sealer`), under a key derived with HKDF from `auth.secret` and a label of its own, so there is no second secret to set and the browser can neither read the verifier nor change the flow.
- **Google** (`openid email`) reads the user from the ID token in the token endpoint's answer. Its signature is not checked: it comes straight from Google over TLS, which OpenID Connect Core 3.1.3.7 accepts instead; issuer, audience, expiry and nonce are. **Discord** (`identify email`) reads `/users/@me`. Both go through `oauth_http`, which answers `ErrProviderRefused` for a 4xx or an answer that does not decode, and a plain error (the error net's `internal`) when the provider could not be asked.
- **`accounts.OutcomeOf` decides, and one verified address is one player.** To sign in: an identity already linked signs in to its account (`SignedIn`); a new one whose verified address another identity already holds joins that account, and the browser moves there (`Joined`, `SIGNED_IN` on the wire); otherwise it links to the account the browser is on (`Linked`); no account, or one that already holds a user of that provider, gets a new account (`Created`). The guest a browser leaves is left as it was — nothing is merged, and the prune deletes it later.
- **The owner of an address is `Store.AccountOfEmail`**: the account of the oldest identity holding it verified, ignoring case (index `identities_email` on `lower(email)`). `signin.Admitter` asks only for a new identity with a verified address, so it covers all three ways in alike: an email code finds a Google or Discord account, and Google or Discord finds an email account or each other. **An owner that already has a user of this provider is not joined** — a second Google on one account is refused everywhere else too — and the old rule applies. Accounts that already hold the same address apart, made before this rule, stay apart: a known identity outranks the address.
- **A link of an address another account holds is refused** (`ErrIdentityLinkedElsewhere`), as a link of an identity another account holds is.
- **A link never moves the browser.** A new identity links to the account; one already on this account is `SignedIn` and changes nothing but the session token; one on **another** account is `ErrIdentityLinkedElsewhere`; an account that already has a user of that provider is `ErrProviderAlreadyLinked`. A refusal writes nothing and keeps `cp_sid`: only `cp_oauth` is cleared. The handler answers `AlreadyExists` with a `LinkRefusal` detail, which is what the client matches. Before this, a link to an identity another account used moved the browser to that account, and the two accounts could never be joined. To move an identity, the player signs in with it, deletes that account, then links it — there is no merge.
- **`accounts.SignIn` is written in one transaction**: the new account if any, the identity if new, the new session, and the deletion of the browser's previous session. The session token changes on every sign-in. Two browsers linking the same new identity at once: the second insert finds it taken (`ErrIdentityTaken`), and the use case runs once more with the identity known — a sign-in moves, a link is refused.
- **An email is kept only when the provider says it is verified** (`accounts.NewIdentity`, `Claim.VerifiedEmail`), and it is `NULL` otherwise. An unverified address finds no account and is found by none.
- **`identities.provider` is the type `identity_provider`** (`google`, `discord`, `email`). A new provider is a migration that adds its value, beside its `signin` constant, its `authv1.Provider` and its name in `me_query`; `TestEveryProviderTheProtoNamesIsAnsweredAsTheOneStartSignInNamed` saves each one through the store. Identities linked at the same time are ordered by `provider::text`, the name, in the store and in `GetMe`: without the cast they come in the order the type declares, and the contract suite fails.
- **Off by default** (`auth.signIn.enabled`). Off, `signin.Providers` is empty and both RPCs answer `Unimplemented`, which Connect sends as HTTP 404. A provider is offered once its `clientId` is set. `StartSignIn` and `CompleteSignIn` spend the mint budget: each can cost a round trip to a third party or make an account.
- **`GetSignInOptions` is how the client knows which buttons to show**: every provider offered, then email when `auth.email.enabled` is on (`signin.Offer`), and an empty list while both are off — never `Unimplemented`, since the question has an answer either way. **It is not throttled and sets no cookie.** A client asks on every page load, and probing with `StartSignIn` instead would spend the mint budget a real sign-in needs and start a flow for nothing. `TestAskingIsNotThrottled` pins it. A server with the whole `auth` module off still 404s it, which the client reads as no provider.
- **The client mints again after `CompleteSignIn`**, so its click token carries the account. The old token names the old account until it expires (1h).
- **`CompleteSignIn` publishes `auth.v1.SignedIn`** once the sign-in is saved: the account the browser was on, empty for none, and the one it is on now. A refusal publishes nothing. `player` moves the roster line, since the client holds no token to announce with until its next click.
- `auth.NewModuleWithFakeProviders` (behind the tag) boots the module with `signin.FakeProvider` for every provider: `Grant(code, claim)`, and the fake checks the verifier against the challenge it was shown. `e2e/sign_in_test.go` drives it.

## Signing in by email (`internal/auth/internal/signin/`)

**A six-digit code, sent by this server through Cloudflare Email Sending, over two RPCs.** No magic link: a mail scanner opens a link before the player does and spends it, and many players read mail on a phone but play on a desktop.

- **`StartEmailSignIn(email, intent, attestation_token)`** reads the address (`signin.AddressOf`), checks a fresh Turnstile token, has `signin.Challenges` issue a `signin.Challenge` — an id, the code, the address, the intent and a link's account — sealed into the `cp_email` cookie (10 minutes, same attributes as `cp_sid`), and gives its letter to `signin.Post`, which refuses a disposable domain, spends the address's budget and mails it. Nothing is stored on the server.
- **The use cases only orchestrate.** The rules are on two domain objects the module builds once and hands to both email use cases: `Challenges` (whether email is offered, issuing and sealing, opening the cookie, spending a guess and checking the code) and `Post` (where a letter may go, and the mailer). `signin.LinkTarget` is the account a link starts on, for both start use cases. Start takes five dependencies, complete three.
- **The address is the identity**: provider `email`, subject the address trimmed and in lower case. An address a Google or Discord identity already verified signs in to that account, and links the email identity there — see `accounts.OutcomeOf` above.
- **`AddressOf` takes one bare address** — no display name, no comment, no quotes, a domain of letters, digits and hyphens with a dot, 254 bytes at most. It refuses with `ErrAddressInvalid`, which the handler answers `InvalidArgument` with an `EmailRefusal` detail, as it does a disposable domain.
- **Disposable domains are refused**, from [disposable-email-domains](https://github.com/disposable-email-domains/disposable-email-domains) (CC0, ~9,000 domains), vendored and embedded in `embedded_disposable_domains` like the VPN lists. A subdomain of a listed domain is refused too. Refresh with `make disposable-domains` and commit; the test asserts the list still loads whole. **This is what keeps an email account worth the linked allowance**: a free inbox for anyone costs a bot nothing.
- **The address is read before Turnstile**, which costs a round trip; **Turnstile comes before the post**, so a caller that proves nothing cannot spend an address's budget and keep its owner from signing in. The IP is passed to Turnstile and is not required: unlike the click token, nothing here is bound to it.
- **Two budgets, both in memory** (`cpratelimit`): per address, three codes at once then one every twenty minutes (`auth.email.sendLimiter`), so nobody fills an inbox that is not theirs; and per challenge, `signin.MaxAttempts` (5) guesses, refilling one per `ChallengeTTL`, so a bucket is swept within an hour. The cookie is replayable — a browser sends the same one with every guess — so the guesses can only be counted on the server. A restart forgets both; it is rare, and costs an attacker's five more guesses at one million codes. Both RPCs also spend the mint budget.
- **The answer is the same whether the address has an account or not**: a known address signs in, a new one is linked or made. Nobody learns which addresses play here.
- **`CompleteEmailSignIn(code)`** opens the cookie, spends a guess, checks the code (constant time) and the expiry, then signs in exactly as `CompleteSignIn` does: **`signin.Admitter` is the one place both paths go through** — the caller's account (`Visitor`), `accounts.OutcomeOf`, `SaveSignIn`, the retry on `ErrIdentityTaken`, and `auth.v1.SignedIn`. A wrong code is `InvalidArgument` and **keeps** `cp_email`, so the player types again; everything else clears it. Spent guesses, a lapsed challenge or no cookie are `FailedPrecondition`: ask for a new code.
- **A code is not burnt by its success.** The cookie is cleared, and a replay needs both the HttpOnly cookie and the code — that is the player who just signed in, and a replay signs them in to the same account again.
- **An email account is linked**: it is never pruned, it refills at the linked multiplier, and it may pick a username.
- **`auth.email.delivery` is `cloudflare` or `log`.** `cloudflare_mailer` posts to the REST API with a token that has *Email Sending: Edit* only; a non-2xx, a body that is not JSON, or a permanent bounce of the address is an error, and nothing is set. `log_mailer` writes the letter, code included, to the server log at `WARN` — the local choice, with no account anywhere. The server warns at boot when it is on.
- `auth.NewModuleWithFakeProviders` turns email on with `signin.FakeMailer` and codes `000001`, `000002`, …; `e2e/email_sign_in_test.go` reads the code out of the letter.

## Operators

**An operator is a game account flagged operator, with a session of its own for the admin page.** Two apps, two
sign-ins, one account base: the admin account is the player account, so `GrantCharges` to its own id is "give me a
bomb to test". `auth.accounts.operator` is a boolean (migration `20261017120000_account_operator`), false for every
account until an operator sets it by hand (`deploy/vps/README.md`, "Make an account an operator"). It is not the
chat crown: `player.profiles.admin` is a display flag, and holding one never grants the other.

- **`adminsignin` is the admin page's sign-in: Google only**, with a Google client of its own (`auth.admin.google`,
  redirect `auth.admin.redirectUrl`). It reuses the game's `signin.Flow`, sealer and Google provider, under a cookie
  of its own (`cp_admin_oauth`), so a game sign-in in progress is never overwritten. **It only finds an account**:
  `Store.Candidate` is the account of the Google identity and whether it operates. It never makes one, links one or
  joins by email, and an identity nobody holds is `ErrUnknownIdentity`, refused as `ErrNotAnOperator`.
- **The admin session is its own concept** (`adminsignin.Session`, table `admin_sessions`, migration
  `20261017130000_admin_sessions`): `Issuer.Session` opens one for an operator only, for `auth.admin.sessionTTL` (12h)
  and never extended. Its cookie, `cp_admin`, is `SameSite=Strict` (`Token.StrictCookie`) and host-only on the API's
  domain. `cp_sid` never opens the admin page, and signing out of one leaves the other. A session goes with its
  account (cascade); one that expired stays until then, a row per sign-in.
- **`auth.v1.AdminService` is that sign-in** (`StartSignIn`, `CompleteSignIn`, `SignOut`) and `WhoAmI`, mounted with
  `props.AdminRPC`: on the console listener, and behind the admin gate. The sign-in spends the mint budget
  (`NewAdminRateLimitInterceptor`). `CompleteSignIn` answers `failed_precondition` to start again and
  `permission_denied` for anybody but an operator, and clears `cp_admin_oauth` on every answer it maps.
- **`InternalService/GetOperator(cookie)` and `WhoAmI` are who a browser is, as an operator**:
  `get_operator_handler/operator_query` asks SQL for a live admin session (`cp_admin`) whose account is flagged, and
  answers its account id. The flag is read on every call, so unsetting it ends a session already open. No admin
  session is `unauthenticated` (`ErrNoSession`); anybody else is `permission_denied` (`ErrNotAnOperator`).
- **The session is read on every call**: no cache. Admin calls are a handful a minute.

## Signing out, deleting, pruning

- **`SignOut`** deletes this browser's session and clears the cookie; a browser with no session succeeds too. **`SignOutEverywhere`** deletes every session of the account and answers `Unauthenticated` with none. Neither touches the account. Both publish **`auth.v1.SignedOut`** once a session was deleted; `SignOut` reads the session first, expired or not, to name its account.
- **`DeleteAccount`** deletes the account row, and its identities and sessions go with it by cascade, then publishes **`auth.v1.AccountDeleted`**. `player` hears it and deletes the profile and the stats. `planet` hears it and takes the account off every take postgres keeps: the takes stay, with their tile, flag and time (see [Durability](durability.md)). The ledger in memory keeps the id until it ages out (72h at most), as do the revert marks; the chat log ages out.
- **No unlink RPC.** When one comes, a linked account must keep its last provider: without one it is a guest holding an email.
- **`prune_guests_usecase`** is an `Executor` and a `Runner` that calls it; `log_prune_guests` is the decorator that logs, so the loop holds no log line. Every `auth.prune.interval` (1h) it deletes, 1000 rows a statement, the accounts with no identity whose `last_seen_at` is older than `auth.prune.idleFor`, and publishes `auth.v1.AccountDeleted` for each (`PruneGuests` answers the ids it deleted). That defaults to `guestTTL`, and less refuses the boot: `last_seen_at` moves when a session is extended, so a guest idle that long has no live cookie left.
