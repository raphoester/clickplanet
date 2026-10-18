# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**The operator's page, at `backoffice.clickplanet.lol`.** It calls the backend's `AdminService`s, which were
reachable only over SSH before, and **every admin method has a place on it**. One page at a time, picked in the
burger menu (see [The menu](#the-menu)), by group:

- **Feedback**: Ideas and bugs, the notes players send, read only.
- **Moderation**: Reports, which the operator acts on and closes (see [Acting on a report](#acting-on-a-report)),
  and Players, who played and every tool on one player (see [The player page](#the-player-page)).
- **Map**: Reassign a country, Paint random tiles, Download a replay (see [The map tools](#the-map-tools)).
- **Ops**: Reconcile titles and Name accounts, the runs after a deploy (see [Ops](#ops)).

What is left of the plan is a live chat view, which no admin method serves.

It is a third app, not a page of the game: **it imports nothing from `apps/frontend`** and has no Three.js (see
the root `CLAUDE.md`, "Independence of the apps").

## Commands

```bash
npm run dev        # Vite on http://localhost:5174: set it as the backend's auth.admin.origin
npm run build      # TypeScript check + Vite bundle → /dist
npm test           # vitest, once (node; a .test.tsx opens with `// @vitest-environment jsdom`)
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

- **The boot needs `auth.admin.google.clientId` and `clientSecret`** once `auth.admin.origin` is set. Made-up
  values boot it, and a session made by hand (below) works with them. **A whole Google sign-in** needs a real
  Google client that lists `http://localhost:5174/auth/callback`, and a Google identity on an account flagged
  operator.
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

## The menu

**A burger at the top left opens a drawer from the left** (`components/Menu`), at every width: every page under
its group, the page shown marked. `MENU` in `navigation.ts` is the only list of pages, and a page's label is its
title (`titleOf`), shown above it. `Console` in `App.tsx` shows the page picked.

- **The drawer is modal**: it takes the focus to the page shown, keeps Tab inside it, and closes on Escape, its
  close button, a click beside it or a page picked, giving the focus back to the burger. The page behind does not
  scroll while it is open.
- **Picking Players while on it goes back to the list.** A report's "Every tool →" goes to Players with its account
  open (`OpenPlayer`).
- **The page picked is not kept** across a reload: the page opens on Ideas and bugs.

## Acting on a report

`src/reports/` is the Reports page: `ReportsView` shows the list, or one report's detail.

- **The list shows the open reports first** (`ListReports` with `open_only`), and a switch shows them all. A
  decided report shows its outcome, comment and time (`DecisionLine`). The list is read again on the way back
  from a detail, so a report just closed leaves the open list.
- **The detail** (`ReportDetail`): who reported whom, each by name and account id, the reason and the text. With
  a `message_id` it reads the message (`GetMessage`, `MessagePanel`): its text, time and country, struck through
  with "Removed at …" once removed. A message past the chat's retention is `not_found`: "This message is no
  longer kept."
- **The actions, all on the reported account id**, never its name: remove the message (`RemoveMessage`, only while
  it is shown), mute (`Mute`, an hour by default), ban (`BanPlayer`, no duration by default, so the antibot's
  ladder picks the next step) and inspect (`InspectPlayer`). Then close the report (`DecideReport`): Acted or
  Dismissed, with an optional comment of up to 1000 characters. A second decision replaces the first.
- **"Every tool →" opens the reported account on the player page** (`OpenPlayer` in `navigation.ts`, which `App`
  provides): unban, revert, its address. The mute, ban and inspect here are the player page's own components.
- **`reports/actions.ts` holds the report's own calls** (remove, decide); the mute, ban and inspect are
  `players/actions.ts`'s.

## Every action, on every page

- **Every action that changes something asks first** (`components/ConfirmAction`): the button opens a question
  with "Yes, …" and Cancel, and only Yes calls the server. Then it says what the server answered ("Muted until
  …"), or the refusal. **A read asks nothing** (inspect, a dry run, a list, a replay): `useAction` runs it on a
  click and holds its answer, as `useCall` does on mount.
- **What can be undone by nothing counts first**: a revert, a reassign and a paint stay disabled until their dry
  run, and the question quotes it ("Give back the 120 tiles Ana holds?"). The dry run holds for the inputs it ran
  on alone: changing a country disables the action again, and **a question whose inputs change under it is taken
  back** (`ConfirmAction` with `disabled`).
- **The refusals in plain words** (`wordsOf` in `refusal.ts`, and each section's `refusalOf` over it, which says
  what its `not_found` and `failed_precondition` mean): `not_found` is a message or a report no longer kept, or no
  ban left to lift; `failed_precondition` is the antibot off (no ban, unban or inspection); `invalid_argument`
  quotes the server (an unknown country, a grant of nothing); `deadline_exceeded` says the call may have done part
  of its work; a session that ended says to sign in again.
- **The long calls take `LONG_CALL_MS`** (`api.ts`, 2 minutes) instead of the transport's 10s: a revert, a
  reassign and a paint are paced on the server, a replay is MBs, and the two ops runs page through every player.
- **Every page takes its clients** (`clients.ts`: `ChatAdmin`, `PlanetAdmin`, `PlayerAdmin`, `SupportAdmin`,
  defaulting to `CLIENTS`), so a test hands it `FakeAdmin` (`fakeAdmin.ts`), which answers every method, records
  each request (`askedFor`) and can refuse a method with a code. The jsdom tests drive each action through its
  page.

## The player page

`src/players/`. **A player is an account or an address** (`Caller` in `caller.ts`), never both: the planet's tools
take either. An account id is a UUID; anything else typed is an address, which the server reads as its scope (the
address over IPv4, the /64 over IPv6) and checks.

- **The list** (`PlayerList`): `TopPlayers` (most takes first, then most tiles held), or **By flag**,
  `FindPlayers` with a flag and an optional ground. 20 by default, up to 500. Each row of the ledger's 72 hours is
  one account on one address, with its takes, the tiles it still holds and their share of its takes (many takes
  and few tiles is a player painted over as fast as it paints), its rates and any ban. The list is read again on
  the way back from a player, as the reports are.
- **Open a player** from a row (its account, or its address when it has none), from the field at the top, or
  from a report. The other half of a row (the address of an account, the account of an address) opens from the
  header, and the inspection's address of an account's latest take opens too: **a guest the jury banned is banned
  on both**, and an unban lifts one key.
- **The page inspects on open** (`InspectPlayer`): the antibot's readings, its ban and its clicks. The antibot off
  reads as such, not as a failure. A ban or an unban reads it again.
- **The tools**: ban (`BanPlayer`, the ladder's next step by default), unban (`UnbanPlayer`, which forgets the
  latest offence; `not_found` is no ban left), revert (`RevertPlayer`, dry run first), and for an account alone
  mute (`Mute`) and charges (`GrantCharges`: a refill, a bomb, enclosures, spread clicks, shields; the answer
  says what the account held before and after, since the caps hold).

## The map tools

`src/map/`, a page each.

- **Reassign** (`ReassignCountry`): every tile one country holds goes to another. The dry run counts both sides.
  A `from_after` above zero is tiles retaken while it ran: the page says to run it again.
- **Paint** (`PaintRandomTiles`): a flag, a ground to start on (empty: the whole map), a count and a proximity
  (0 scattered, 1 one patch). The dry run draws without painting, so the real run draws again: the question says
  "about".
- **A country is typed by its id** (`CountryField`), and the page names it below ("dz" → Algeria) with
  `Intl.DisplayNames`, plus the game's own ids no standard names (`countries.ts`: England, Scotland, Wales,
  Brittany). It is a hint: the server holds the list of countries and refuses any other.
- **Replay** (`GetReplay`): a span back from an end (empty: now), saved as `replay-<since>-<until>.json`, the
  answer as protobuf JSON, which the frontend's `npm run clip` reads as it reads what `clips fetch` saves. The
  ledger keeps 72 hours. The file is made in memory (`download.ts`, a `blob:` link), which the CSP allows.

## Ops

`src/ops/`, a page each: `ReconcileTitles` (`TitlesPanel`, after a deploy that adds a title or changes a rule) and
`NameAccounts` (`NamesPanel`, once, after the deploy that brought generated names). Each asks first, and each is
harmless twice.

## Deploy

**A Cloudflare Worker with static assets, `clickplanet-backoffice`** (`wrangler.jsonc`), like the game's. Its
custom domain `backoffice.clickplanet.lol` is set on the Worker in the dashboard, not in `wrangler.jsonc`: a route
there would make the DNS record on the first deploy.

**Workers Builds deploys it from `main`**, on a push that changes something under `apps/backoffice/`: `npm ci && npm
run build`, then `npx wrangler deploy`, with `NODE_VERSION` 22 and `VITE_API_BASE_URL` set on the build. A change
to `proto/` alone ships nothing: the generated code is committed.

`public/_headers` is the page's security:

- **`Content-Security-Policy`**: scripts, styles and images from the page alone, calls to
  `https://api.clickplanet.lol` alone, and no frame (`frame-ancestors 'none'`): the pages ban, revert and repaint
  the map, and a framed page could be clicked through. **A new API host is a change here.**
- `Referrer-Policy: no-referrer` and `X-Robots-Tag: noindex`.
- `/assets/*` is immutable, `/index.html` is revalidated.

## Layout

```
src/
  api.ts            the transport, one client per AdminService, LONG_CALL_MS
  clients.ts        the narrow client types every page takes, and CLIENTS
  fakeAdmin.ts      FakeAdmin, every admin method for the tests
  refusal.ts        a page's error → signedOut, notOperator, startAgain or failed; an action's in words (wordsOf)
  useCall.ts        one call's state, run on mount
  useAction.ts      one read's state, run on a click
  navigation.ts     MENU, the pages by group, and OpenPlayer
  countries.ts      a country id's name
  App.tsx           the callback, or WhoAmI first, then the page the menu picked
  signIn/           callbackOf, SignInButton, SignInCallback, SignOutButton
  components/       Menu, ConfirmAction, FailureNotice, AccountId and Copyable (with Copy), CountryField, Fact,
                    Person, When
  support/          Ideas and bugs: NoteList, labels.ts (names, outcomes and the device lines)
  reports/          Reports: list, detail, the report's own actions
  players/          Players: list, PlayerView, the tools on one player, actions.ts
  map/              Reassign a country, Paint random tiles, Download a replay
  ops/              Reconcile titles, Name accounts
  gen/grpc/         generated from ../../proto, committed
```

**An account is shown by its id, with Copy**, beside its name: a name can change hands, and the id is what the
operator tools take.

Plain CSS in `src/index.css`, tokens on `:root`, dark from `prefers-color-scheme`. **It must work on a phone**:
16px gutter, and nothing may scroll sideways, so every id and text wraps (`overflow-wrap: anywhere`).
