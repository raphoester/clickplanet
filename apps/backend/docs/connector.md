# Claude as a connector

**Claude reaches the admin services as a claude.ai custom connector.** The operator adds
`https://api.clickplanet.lol/mcp` once (claude.ai, Settings, Connectors), signs in on the admin page and clicks
Allow. The connector then works everywhere the operator uses Claude: claude.ai, the iPhone app, and Claude Code,
which shows the claude.ai connectors. There is no key, no token to paste and no script. The tokens renew themselves.

Two parts: the `mcp` module serves the tools, and auth serves the OAuth flow and the tokens. Both are on the admin
listener. Caddy sends `/mcp`, `/oauth/*` and `/.well-known/oauth-*` there with no CORS: they are plain handlers of
that listener (`props.AdminHTTP`), not admin services, so they are outside the gate's middlewares.

## The tools

- **The tools are the admin services, read from their protos.** `make proto` writes
  `generated/descriptors/admin.binpb`, a descriptor set of every `admin.proto` **with its comments**, and the module
  embeds it. `tools.CatalogOf` makes one tool of each method: `<context>_<method in snake case>`
  (`planet_grant_charges`), its leading comment as the description, its request as the input schema (protojson's
  JSON names, an enum by name, a timestamp in RFC 3339, a duration as `"3600s"`, each field's comment as its
  description), and `readOnlyHint` when the method is `NO_SIDE_EFFECTS`, `destructiveHint` otherwise. **A new admin
  method is a new tool after `make proto`, with no code here.** Its comment is the text Claude reads, which is why
  admin rpcs keep a comment (see [`docs/proto.md`](../../../docs/proto.md)).
- **Left out:** `auth.v1` (an app never signs in to the page or lets an app in), every method marked
  `deprecated = true` (its replacement is a tool already), and the hidden list, `tools.Hidden` in `internal/mcp/internal/tools/tools.go`:
  - reads too big for a tool's answer: `planet` `GetReplay` (megabytes for a day), `GetLedger` (72 h of every act,
    page by page: a 24 h poll of it once pushed the box into swap) and `GetTerritories` (every tile of every country);
  - the clips job's writes, which only the job calls over ssh: `clips` `KeepSnapshot`, `MakeCast` and
    `KeepAppearance`;
  - the posting of a job's video, which only a job calls over ssh: `publishing` `CreateMediaUpload` (a link to send
    a file to, which a tool has no file to send) and `SchedulePost` (a public post on TikTok and YouTube, at a slot
    `PlanPost` gave), and `promo` `PostTeaser` (a battle's teaser posted, which only the teasers job calls, with the
    files it uploaded). `publishing` `GetRoom`, `PlanPost` and `ListPosts` stay: what each platform has left today,
    when a post would go out, and what was posted; and `promo` `ListDueTeasers`, the battles whose teaser is due.
- `tools_test.go` fails on an admin method with no comment, and on an embedded set that does not match the
  generated code.
- **The server does not decide what Claude may do without asking.** `readOnlyHint` is a hint: the operator sets the
  writes to "Needs approval" on claude.ai, and Claude Code asks for each tool not on its allow list.

## A call

- **A call goes through the gate like the page's.** `tools/call` posts the arguments as they are, as protojson, to
  the admin listener (`props.Admin.Dial()`, `connect_admin_caller`) with the app's token. The gate admits it as the
  operator and writes the `admin call` line with `client=Claude`, the audit decorators log what it did, and the
  error net strips an internal cause. **What Claude does is kept apart from what the operator does**: the gate hands
  the app's name to the module (`cpctx.Operator`), and a mute, an unmute, a removal and a rename keep it with their
  reason (`chat_list_moderation`, `player_list_renames`: see [The moderation history](chat.md#the-moderation-history)).
  A refusal (a Connect code and its sentence) is the tool's result with
  `isError`, so Claude reads it and tries again. Only a tool nobody offers is a JSON-RPC error. A call has 5 minutes:
  a reassign or a reconciliation walks the whole map or every account.
- **MCP over Streamable HTTP, answered with plain JSON** (`mcpcontroller`): `initialize` (2025-06-18, 2025-03-26 or
  2024-11-05, the newest for any other), `ping`, `tools/list`, `tools/call`. A notification is a 202, and a batch is
  answered for its requests. No session id and no stream from the server: `GET` is a 405. The handshake carries
  `tools.Instructions`, what Claude is told before any tool: dry runs first, account ids over scopes, the formats.
- **Every MCP request needs a live token.** None is a 401 with `WWW-Authenticate: Bearer resource_metadata="…"`, the
  start of the flow. A lapsed one is a 401 with `error="invalid_token"`, which sends Claude to refresh. An account no
  longer flagged is a 403. The module asks `auth.v1.InternalService/GetTokenOperator` itself
  (`rpc_token_holders`), and the gate asks again on each call. So `mcp` is the one module besides `player` and
  `alerts` that speaks auth's language: it is no game module (`cmd/api/modules_test.go`).

## The gate and a bearer token

- **A call with `Authorization: Bearer` is judged by the token alone**, from any origin and with no cookie read. A
  browser never sends a bearer token on its own, so the origin check, which keeps other pages from riding the
  cookie, has nothing to keep out. `admit_usecase` asks `GetTokenOperator` (`rpc_operators`): no live token is
  `unauthenticated`, an account no longer flagged `permission_denied`.
- **A token never makes the page's own calls.** The sign-in, `WhoAmI`, `DescribeConnection` and
  `ApproveConnection` are `permission_denied` before auth is asked, so a connected app can never let another app in.
- **`GetTokenOperator(token)`** (`token_operator_query`) answers a live access token whose account is flagged, and
  the app's name. The flag is read on every call, so unsetting it ends every connection of the account at once.

## The OAuth 2.1 authorization server

Auth's `connections` (the rules) and `oauthcontroller` (the plain HTTP edge), on the admin listener.

- **Discovery:** `/.well-known/oauth-protected-resource/mcp` (RFC 9728, served by `mcp`) names the API's origin as
  the server, and `/.well-known/oauth-authorization-server` (RFC 8414) lists its endpoints.
- **`POST /oauth/register`** (RFC 7591): any app may register, **only for a redirect in
  `auth.connector.redirectUris`**, which is Claude's two callbacks unless set. So a code can only ever go to Claude,
  whoever clicks Allow. Every client is public: PKCE (S256, required) proves a code is its own, and a secret sent
  anyway is ignored. Claude Code's own OAuth (a localhost callback) is not allowed on purpose: any local program
  could then receive a code.
- **`GET /oauth/authorize`** checks the client and its redirect, then sends the browser to the admin page's
  `/connect` with the request. A client or a redirect this server did not register is a 400 page, never a redirect.
- **The page** shows who asks and where it will be sent back to (`DescribeConnection`). Allow calls
  `ApproveConnection`, which answers the app's callback with a code, the state and `iss` (RFC 9207). Deny sends the
  browser back with `access_denied`. `audit_approve_connection` logs every approval at Warn. The page is the
  backoffice's ([Connecting Claude](../../backoffice/docs/connect.md)).
- **`POST /oauth/token`** trades the code once, within a minute, for the client and the redirect it was issued to,
  with the verifier. It answers a pair: an access token for an hour (`cpat_…`) and a refresh token for 90 days
  (`cprt_…`). **A refresh token is good once**: each trade spends it and answers a new pair of the same connection, so
  a connection used once a season never lapses. One sent by another client is spent and buys nothing.
- **A spent refresh token traded again ends its whole connection** (`admintokens.ErrReused`): somebody holds a copy,
  so every token of the connection lapses, the thief's pair and the app's alike, and the operator connects again.
  The rule is `Issuer.Trade`, over every token of the connection, read in one transaction with `FOR UPDATE`, so two
  trades of one token never both win. A client that retries a refresh it already won ends its own connection too.

## What is kept

- **Nothing of the flow is stored but the tokens.** A client id, a request and a code are each sealed with
  AES-256-GCM under a key of their own derived from `auth.secret` (`aes_connection_sealer`), so a sealed client never
  opens as a code. A spent code is kept in memory until it lapses (`inmemory_spent_codes`).
- **The tokens are `auth.admin_tokens`** (migration `20261105120000_admin_tokens`): the SHA-256 of each, its kind
  (the enum `admin_token_kind`: `access`, `refresh`, or `spent`, a refresh token traded), the account (cascade), the
  app's client id hash and name, the connection it belongs to (one code exchange starts one), and when it ends.
- **The store only finds and saves**: `Connection(hash)` reads every token of the connection a hash belongs to, and
  `SaveTokens` upserts snapshots in one statement. Spending and ending are the domain's (`Issuer.Trade`). Nothing is
  pruned: a lapsed row is refused anyway, and one operator makes a few dozen rows a day.
- **Ending a connection:** take the account's operator flag away (every connection at once, and the backoffice
  too), or delete its rows of `auth.admin_tokens` (`deploy/vps/README.md`, "Connect Claude"). The page has no list of
  connections.

## Configuration

- `auth.connector.enabled`, `auth.connector.issuer`: the authorization server, and the API's public origin it is
  reached on (`https://api.clickplanet.lol`, from `API_DOMAIN` in production). It needs `auth.admin.origin`, where
  the operator lets the app in. An issuer that is not an origin refuses the boot.
- `auth.connector.redirectUris`: where an app may be sent back to with a code. Claude's two callbacks unless set.
- `mcp.enabled`, `mcp.url`: the MCP endpoint (`https://api.clickplanet.lol/mcp`). Its origin is the authorization
  server's, so it must be `auth.connector.issuer`. It needs the admin and internal listeners, or the boot is refused.

## Known risk

**Cloudflare is in front.** Claude calls from Anthropic's servers. A bot rule or a challenge on `api.clickplanet.lol`
would answer it a page instead of JSON. If adding the connector fails before a sign-in page shows, look in Security,
Events, and skip the rule for `/mcp`, `/oauth/*` and `/.well-known/*`.

## Tests

`e2e/connector_test.go` is one smoke test that walks the flow as Claude does, from the URL alone: the 401, both
metadata documents, a registration, the operator's yes, the code traded once, `initialize`, `tools/list`, a call that
grants the operator a bomb, a refusal as the tool's result, a token refused on `ApproveConnection`, a refresh, and the
flag taken away. Every refusal path is in the unit tests of its package.
