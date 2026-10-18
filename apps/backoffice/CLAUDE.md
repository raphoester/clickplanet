# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**The operator's page, at `backoffice.clickplanet.lol`.** It calls the backend's `AdminService`s, which were
reachable only over SSH before. Today it has one page, read only: the support notes (ideas and bugs) and the
player reports, newest first. The plan after it: a player page (search, ban, mute, revert, grant charges), map
tools, a live chat view, then ops pages.

It is a third app, not a page of the game: **it imports nothing from `apps/frontend`** and has no Three.js (see
the root `CLAUDE.md`, "Independence of the apps").

## Commands

```bash
npm run dev        # Vite on http://localhost:5174: set it as the backend's auth.adminGate.origin
npm run build      # TypeScript check + Vite bundle → /dist
npm test           # vitest, once
npm run lint       # ESLint
npm run proto      # Regenerate src/gen/grpc from the shared ../../proto
```

`.github/workflows/check-backoffice.yml` runs lint, build and tests on every PR touching this app or `proto/`.

## How it reaches the API

- **Every call goes to `https://api.clickplanet.lol/<package>.v1.AdminService/<Method>`** (`src/api.ts`).
  Caddy sends those paths to the backend's admin listener (the backend's `CLAUDE.md`, "Operator tools").
  `VITE_API_BASE_URL` points it elsewhere; it is a build variable.
- **The page signs in on its own, with Google only** (`auth.v1.AdminService`, the backend's `CLAUDE.md`,
  "Operators"). Its session is the `cp_admin` cookie: 12 hours, never extended, `SameSite=Strict`, host-only on
  `api.clickplanet.lol`. It travels from `backoffice.clickplanet.lol` because both are one site, and the transport sends
  `credentials: "include"`. **The game's `cp_sid` never opens this page**, and signing out of one leaves the other.
- **The sign-in** (`src/signIn/`): "Sign in with Google" calls `StartSignIn` and sends the browser to the
  `authorization_url` it answers. Google comes back to `/auth/callback` (the backend's `auth.admin.redirectUrl`).
  `main.tsx` reads the code and the state there (`callbackOf`) and **takes them out of the address bar before
  anything renders**: a code is good once. `SignInCallback` calls `CompleteSignIn` once (a ref guards it, since
  StrictMode runs an effect twice), then shows the page. A Google refusal, or a link with a part missing, offers
  the button again. The Worker serves `/auth/callback` through its SPA fallback.
- **Sign out** is a button beside the account id (`SignOut`), and then the page asks `WhoAmI` again.
- **The gate admits a session of an account whose `auth.accounts.operator` is true**, and only from the `Origin`
  `https://backoffice.clickplanet.lol`. The flag is set by hand (`deploy/vps/README.md`, "Make an account an
  operator"); it is read on every call, so unsetting it ends a session already open.
- **The refusals** (`refusal.ts`, `FailureNotice`): `unauthenticated` is "Sign in to continue", `permission_denied`
  is "This Google account is not an operator" (a wrong origin too), `failed_precondition` from `CompleteSignIn` is
  "The sign-in expired. Start again". Each offers the Google button. Anything else is "The call failed", with Try
  again. The server's message is shown under each, for the operator.
- **`auth.v1.AdminService/WhoAmI` says who the page is**: the operator's account id, from the `cp_admin` session.
  `App` asks it first, and shows nothing else until it answers.
- **`useCall`** runs one call, aborts it when the component goes or the call changes, and holds `loading`,
  `done` or `failed`. Its `run` must keep its identity (module scope or `useCallback`).

## Running it locally

The page needs a backend with auth on and an admin listener. In a copy of `apps/backend/cmd/api/example.yaml`:
`httpServer.adminBindAddress: 127.0.0.1:8083`, `auth.admin.origin: http://localhost:5174`, `auth.enabled: true` and
an `auth.secret` (`openssl rand -hex 32`). Postgres on `localhost:5432`. Then
`VITE_API_BASE_URL=http://localhost:8083 npm run dev`.

- **A whole Google sign-in** needs `auth.admin.google.clientId` and `clientSecret` of a Google client that lists
  `http://localhost:5174/auth/callback`, and a Google identity on an account flagged operator. Without a client,
  `StartSignIn` is `unimplemented`.
- **Without Google**, open a session by hand. The table keeps the SHA-256 of the cookie's value, so pick a value,
  insert its hash, and set the cookie on `http://localhost:5174` (a cookie ignores the port, so it reaches `:8083`;
  the backend does not see that it is not `HttpOnly`):

  ```sql
  UPDATE auth.accounts SET operator = true WHERE id = '<account id>';
  INSERT INTO auth.admin_sessions VALUES (sha256('local-test'), '<account id>', now(), now() + interval '12 hours');
  ```

  ```js
  document.cookie = "cp_admin=local-test; path=/"
  ```

  Any account row will do, a guest's included (`curl` a `CreateSession` against `:8080`): only the Google
  sign-in checks the identity, and this skips it.

## Deploy

**A Cloudflare Worker with static assets, `clickplanet-backoffice`** (`wrangler.jsonc`), like the game's. Its
custom domain `backoffice.clickplanet.lol` is set on the Worker in the dashboard, not in `wrangler.jsonc`: a route
there would make the DNS record on the first deploy.

**Workers Builds deploys it from `main`**, on a push that changes something under `apps/backoffice/`: `npm ci && npm
run build`, then `npx wrangler deploy`, with `NODE_VERSION` 22 and `VITE_API_BASE_URL` set on the build. A change
to `proto/` alone ships nothing: the generated code is committed.

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
  refusal.ts        a call's error → signedOut, notOperator, startAgain or failed
  useCall.ts        one call's state
  App.tsx           the callback, or WhoAmI first, then the page
  signIn/           callbackOf, SignInButton, SignInCallback, SignOutButton
  components/       FailureNotice, AccountId (with Copy), When
  support/          the support page: NoteList, ReportList, labels.ts (names and the device lines)
  gen/grpc/         generated from ../../proto, committed
```

**An account is shown by its id, with Copy**, beside its name: a name can change hands, and the id is what the
operator tools take.

Plain CSS in `src/index.css`, tokens on `:root`, dark from `prefers-color-scheme`. **It must work on a phone**:
16px gutter, and nothing may scroll sideways, so every id and text wraps (`overflow-wrap: anywhere`).
