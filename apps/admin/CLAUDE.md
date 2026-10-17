# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**The operator's page, at `admin.clickplanet.lol`.** It calls the backend's `AdminService`s, which were
reachable only over SSH before. Today it has one page, read only: the support notes (ideas and bugs) and the
player reports, newest first. The plan after it: a player page (search, ban, mute, revert, grant charges), map
tools, a live chat view, then ops pages.

It is a third app, not a page of the game: **it imports nothing from `apps/frontend`** and has no Three.js (see
the root `CLAUDE.md`, "Independence of the apps").

## Commands

```bash
npm run dev        # Vite on http://localhost:5174, the backend's httpServer.operatorOrigin
npm run build      # TypeScript check + Vite bundle → /dist
npm test           # vitest, once
npm run lint       # ESLint
npm run proto      # Regenerate src/gen/grpc from the shared ../../proto
```

`.github/workflows/check-admin.yml` runs lint, build and tests on every PR touching this app or `proto/`.

## How it reaches the API

- **Every call goes to `https://api.clickplanet.lol/<package>.v1.AdminService/<Method>`** (`src/api.ts`).
  Caddy sends those paths to the backend's operator listener (the backend's `CLAUDE.md`, "Operator tools").
  `VITE_API_BASE_URL` points it elsewhere; it is a build variable.
- **The game's `cp_sid` cookie is the only credential.** The transport sends `credentials: "include"`. The cookie
  is host-only on `api.clickplanet.lol`, and it travels from `admin.clickplanet.lol` because both are one site.
  There is no second sign-in: an operator signs in on the game.
- **The backend admits an account whose `auth.accounts.role` is `operator`**, and only from the `Origin`
  `https://admin.clickplanet.lol`. The role is set by hand (`deploy/vps/README.md`, "Make an account an operator").
- **Two refusals have their own words** (`refusal.ts`, `FailureNotice`): `unauthenticated` is "Sign in on
  clickplanet.lol" with a link, `permission_denied` is "This account is not an operator". Anything else is "The
  API did not answer". The server's message is shown under each, for the operator. A wrong origin is also
  `permission_denied`.
- **`auth.v1.AdminService/GetOperator` says who the page is**: the account id the guard admitted. `App` asks it
  first, and shows nothing else until it answers. The public `AuthService/GetMe` is not an option: the public API
  answers CORS for the game's origin alone.
- **`useCall`** runs one call, aborts it when the component goes or the call changes, and holds `loading`,
  `done` or `failed`. Its `run` must keep its identity (module scope or `useCallback`).

## Running it locally

The page needs a backend with auth on and an operator listener:

1. In a copy of `apps/backend/cmd/api/example.yaml`: `httpServer.operatorBindAddress: 127.0.0.1:8083`,
   `auth.enabled: true`, an `auth.secret` (`openssl rand -hex 32`) and `auth.email.enabled: true`. Email
   delivery `log` puts the sign-in code in the server log. Postgres on `localhost:5432`.
2. Sign in with an email in the game (`apps/frontend`, `VITE_API_BASE_URL=http://localhost:8080`). A guest is never
   an operator.
3. `UPDATE auth.accounts SET role = 'operator' WHERE id = '<account id>'` in psql.
4. `VITE_API_BASE_URL=http://localhost:8083 npm run dev`. A cookie ignores the port, so the game's cookie for
   `localhost` reaches `:8083`.

Without the game: mint and sign in with `curl` against `:8080` (`CreateSession`, `StartEmailSignIn`,
`CompleteEmailSignIn`, with a cookie jar), then set the jar's `cp_sid` with `document.cookie` on
`http://localhost:5174`. The backend reads the header and does not see that it is not `HttpOnly`.

## Deploy

**A Cloudflare Worker with static assets, `clickplanet-admin`** (`wrangler.jsonc`), like the game's. Its
custom domain `admin.clickplanet.lol` is set on the Worker in the dashboard, not in `wrangler.jsonc`: a route
there would make the DNS record on the first deploy.

`public/_headers` is the page's security:

- **`Content-Security-Policy`**: scripts, styles and images from the page alone, calls to
  `https://api.clickplanet.lol` alone, and no frame (`frame-ancestors 'none'`): the later pages ban and revert,
  and a framed page could be clicked through. **A new API host is a change here.**
- `Referrer-Policy: no-referrer` and `X-Robots-Tag: noindex`.
- `/assets/*` is immutable, `/index.html` is revalidated.

## Layout

```
src/
  api.ts            the transport and one client per AdminService
  refusal.ts        a call's error → signedOut, notOperator or failed
  useCall.ts        one call's state
  App.tsx           GetOperator first, then the page
  components/       FailureNotice, AccountId (with Copy), When
  support/          the support page: NoteList, ReportList, labels.ts (names and the device lines)
  gen/grpc/         generated from ../../proto, committed
```

**An account is shown by its id, with Copy**, beside its name: a name can change hands, and the id is what the
operator tools take.

Plain CSS in `src/index.css`, tokens on `:root`, dark from `prefers-color-scheme`. **It must work on a phone**:
16px gutter, and nothing may scroll sideways, so every id and text wraps (`overflow-wrap: anywhere`).
