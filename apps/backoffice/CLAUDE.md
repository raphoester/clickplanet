# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**The operator's page, at `backoffice.clickplanet.lol`.** It calls the backend's `AdminService`s, which were
reachable only over SSH before, and **every admin method has a place on it**. One page at a time, picked in the
burger menu (see [The menu](#the-menu)), by group:

- **Overview**: Home, how the game is doing now and over the last hours (see [The home page](#the-home-page)).
- **Feedback**: Ideas and bugs, the notes players send, read only, each with who sent it (see
  [Who sent a note](#who-sent-a-note)) and its photos and videos (see
  [A note's photos and videos](#a-notes-photos-and-videos)).
- **Moderation**: Reports, which the operator acts on and closes (see [Acting on a report](#acting-on-a-report)),
  and Players, who played and every tool on one player (see [The player page](#the-player-page)).
- **Map**: Reassign a country, Paint random tiles, Download a replay (see [The map tools](#the-map-tools)).
- **Clips**: Clip stats, the daily numbers of the clips the clips job posts, and a CSV of them (see
  [Clip stats](#clip-stats)).
- **Ops**: Reconcile titles and recount bombs and fortifies, the runs after a deploy, and Game rules, the rule set in
  force and the fills of every player's charges (see [Ops](#ops)).

What is left of the plan is a live chat view: `ListMessages` pages through the chat (the report and player pages
read it), and no admin method streams it.

It is a third app, not a page of the game: **it imports nothing from `apps/frontend`** and has no Three.js (see
the root `CLAUDE.md`, "Independence of the apps").

## Commands

```bash
npm run dev        # Vite on http://localhost:5174: set it as the backend's auth.admin.origin
npm run build      # TypeScript check + Vite bundle → /dist
npm test           # vitest, once (node; a .test.tsx opens with `// @vitest-environment jsdom`)
npm run lint       # ESLint
npm run proto      # Regenerate src/gen/grpc from the shared ../../proto
npm run map        # Copy the tile coordinates from the shared ../../map into src/gen/map
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

**A burger at the top left opens a drawer from the left** (`components/Menu`) below 1024px: every page under its
group, the page shown marked. **From 1024px the same list stays open in a sidebar** (`Sidebar`), with no burger,
so a page is one click away (`useWide`, which follows the window as it is resized). `MENU` in `navigation.ts` is
the only list of pages, and a page's label is its title (`titleOf`), shown above it. `Console` in `App.tsx` shows
the page picked.

- **The sidebar stays in place while the page scrolls**, and is not modal: nothing traps the focus or closes it.
- **The drawer is modal**: it takes the focus to the page shown, keeps Tab inside it, and closes on Escape, its
  close button, a click beside it or a page picked, giving the focus back to the burger. The page behind does not
  scroll while it is open.
- **Picking Players while on it goes back to the list.** A report's "Every tool →" and a note's "Open player" go to
  Players with its account open (`OpenPlayer`).
- **The page shown is in the address bar** (`#/reports`, `usePage`): a reload or a link opens it again, and Back
  goes to the page before. An address that names no page opens Home. Only the page is kept, not the
  player open on it.

## The home page

`src/home/`, the page the backoffice opens on. **The backend sends what happened and what is, never a chart**: the
page counts, charts and draws on its own side, so a new view of the game needs no new endpoint. It reads, each
failing alone:

- `planet.v1.AdminService/GetLedger` for the span: every change the live stream showed, as the `TileUpdate` it sent,
  with when and who (the backend's `CLAUDE.md`, "GetLedger"). It is asked for whole slots, from a step back to now
  (`windowOf`, both bounds sent): the last one starts on the step before now, and is still filling. The page counts
  from the window it asked for. `LONG_CALL_MS`: three days is a
  few MB compressed.
- `GetTerritories` for the map now: the tiles each country holds.
- `player.v1.AdminService/GetRoster` for who plays now.
- `ListReports` (open only) and `ListNotes` (the last 24 hours) for what waits for the operator. Up to 500 of each
  are counted, and 500 reads "500+".
- **The tile coordinates** (`places.ts`), the map's own blob from `/map` (`npm run map` copies it into `src/gen/map`,
  and Vite bundles it under `/assets/`): where each tile is, read once a page load, about 5 MB. Its latitude and
  longitude are read as the map's generator reads a vertex: `asin(y)` and `atan2(z, -x)`. The CSP's `connect-src`
  holds `'self'` for it.

What the page does with them:

- **A span is picked above everything it scopes**: 6 hours in slots of 5 minutes, 24 hours (first) in 15, or 3 days
  in hours. Every minute the page reads it all again (`REFRESH_MS`), and Reload does now. **The last answer stays
  shown, dimmed, while the next is read** (`useLast`), so a refresh never empties the page.
- **The counts** (`ledger.ts`): a click is a change marked `clicked` (a take or a strike); a tile changed hands when
  its flag after is not its flag before; a player is who changed one (`by`, compared and never read), an account on
  every address, or an address with no account.
- **The tiles**: who plays now (and how many signed in), the clicks and the players of the span, the open reports and
  the notes of the day. The last two go to their page.
- **Two charts of one series each** (`ActivityCharts`): the clicks a slot, and the players taking tiles a slot. Never
  one chart with two scales. A hover on either reads the same slot on both, in the line above them; the peak is
  labelled; the slot still filling is a hollow dot, off the line, since its count is short. "Table" lists every slot.
- **The heatmap** (`Heatmap`, `heat.ts`): the map seen flat, 1440×720. Each tile is drawn where it is, as wide as
  its longitude stretches far north; what changed hands glows over the land around it (a box blur, twice, round the
  world from east to west), on a log scale up to the hottest pixel. The colors are tokens (`--map-*`, `--heat-0` to
  `--heat-3`), read when it draws, and it draws again when the color scheme changes. The line under it names the
  degree with the most changes, or the degree under the pointer, with its tiles.
- **The countries** (`CountryTable`): what each holds now, its share of the map, and the tiles it won and lost by play
  in the span. Held, Won and Lost order them, and **the bars are a histogram of the column picked**. The ten first,
  then all.
- **Playing now** (`PlayingNow`): each signed-in player, which opens it on the player page (`OpenPlayer`), the guests
  as a count, and the flags they play for.
- **What it does not show yet**: HTTP requests. The clicks come from the ledger, which keeps 72 hours and survives a
  restart; the backend's request counters are in-process, and the metrics poller's history is files on the server.

## Who sent a note

**A note is sent from an account, always**: `SuggestFeature` and `ReportBug` refuse a caller with no session, and
`ListNotes` answers its `account_id`. It is blank only once the account is deleted (the backend anonymizes the
note and keeps its text).

- **"From" is the sender's name and account id, with Copy** (`Person`). `ListNotes` carries no name, so `NoteList`
  asks `player.v1.AdminService/GetAuthors` for the accounts of the page (`namesOf` in `players/actions.ts`, which
  the player list's `withAuthors` uses too). **Names that cannot be read leave the notes listed**, every sender
  "no name", with the reason above them.
- **"Open player" opens the sender on the player page** (`OpenPlayer`): its profile, its inspection, and from
  there the address of its latest take. A note with no account has no button.

## A note's photos and videos

`Attachments` shows each file a note carries, under its text, from the `url` `ListNotes` answers.

- **A photo is a thumbnail that opens the file**, a video plays in place (`preload="metadata"`), and every file has
  its type, its size and an Open link.
- **A file the browser cannot show** (HEIC outside Safari, a QuickTime video Chrome cannot decode) or a link past
  its hour says so, and keeps the Open link. The links are signed for an hour (`support.attachments.linkTTL`):
  Reload signs new ones.

## Acting on a report

`src/reports/` is the Reports page: `ReportsView` shows the list, or one report's detail.

- **The list shows the open reports first** (`ListReports` with `open_only`), and a switch shows them all. A
  decided report shows its outcome, comment and time (`DecisionLine`). The list is read again on the way back
  from a detail, so a report just closed leaves the open list.
- **The detail** (`ReportDetail`): who reported whom, each by name and account id, the reason and the text. With
  a `message_id` it reads the message in its conversation (`GetMessage`, `MessagePanel`), struck through with
  "Removed at …" once removed. A message past the chat's retention is `not_found`: "This message is no longer
  kept."
- **The conversation** (`Conversation`): a message alone cannot be judged, so the page pages through the chat
  around it with `ListMessages`, the message's id as the cursor: the 10 messages before it and the 10 after
  (`FIRST_PAGE`), at once, then 20 more each time "Show earlier" or "Show later" is pressed (`NEXT_PAGE`), from
  the oldest or the newest message shown. A page shorter than asked is the end, and its button goes. The reported
  message shows while they load, marked ("Reported message", `aria-current`), and stays shown if they fail.
  `GetMessage` names no author, so its line takes the name the report gives its account.
- **What the reported account said, in its conversations** (`chat/TheirMessages`, "What Ana said"), for every
  report: a report from a player's card names no message, and one line cannot be judged alone. `ListMessages` with
  the account's id reads its messages, 20 at a time (`THEIR_PAGE`), then `moreConversations`
  (`theirConversations.ts`) reads the whole chat around each, `AROUND` (5) messages each side, and shows them at
  once, newest first, three at a time (`CONVERSATIONS`), "Show more conversations" for the next. **One of their
  messages a conversation already holds gets none of its own, and a conversation that reaches the next one joins
  it**, so an exchange of three insults is one conversation, read with two calls. Their lines are marked
  (`line-theirs`) and tagged `reported`, the reporter's `reporter`. Each conversation grows with "Show earlier" and
  "Show later". Nothing kept is "Nothing they said in the chat is kept."
- **Each line** has its author's name as the chat names it now, its country and its time (with the date when it
  is not the reported message's day), what it answers ("In reply to …"), and is struck through with when it was
  removed. **What the reporter and the reported account said is tagged** `reporter` and `reported`, by account
  id. An author's name opens it on the player page (`OpenPlayer`), so a third player in the conversation is a
  click away; the player already open there stays open, with its row. A page past the chat's retention is
  `not_found`: "That part of the chat is no longer kept."
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
  `FindPlayers` with a flag and an optional ground, or **By name**. 20 by default, up to 500. Each row of the ledger's 72 hours is
  one account on every address it took from, shown with the address of its latest take (or one address, for takes
  with no account), with its takes, the tiles it still holds and their share of its takes (many takes
  and few tiles is a player painted over as fast as it paints), its rates and any ban. The list is read again on
  the way back from a player, as the reports are.
- **By name** (`NameResults`) is `player.v1.AdminService/SearchPlayers` with what the operator typed: every account
  whose username or guest code holds it, any case, the same name first. It reaches past the ledger's 72 hours, so a
  banned player the lists no longer show is found by the name the chat gives it. A row is its name and whether it is
  a username or a guest code, and opens its account (`MatchCard`); it carries no takes, since the ledger is not
  asked.
- **A row is shown by its flag and its name, never its account id** (`PlayerName`): the flag it took the most tiles
  for (`flag_country_id`), as an emoji and a name, then its username or its guest code, then its address. The
  planet's rows carry no name, and planet may not ask `player` (the backend's `CLAUDE.md`, "Player"), so
  the page asks `player.v1.AdminService/GetAuthors` for the accounts of the page once the list answers
  (`withAuthors`). An account the game never named is "Unnamed": it never joined the roster or posted, and the
  game's page joins the roster as soon as it may click, so it is most likely a script. A row with no account is "No account". **Names
  that cannot be read leave the list listed**, every account "Unnamed", with the reason above it.
- **Open a player** from a row (its account, or its address when it has none), from the field at the top, from a
  report, or from a note. The other half of a row (the address of an account, the account of an address) opens from the
  header, and the inspection's address of an account's latest take opens too: **a guest the jury banned is banned
  on both**, and an unban lifts one key.
- **An account's page reads its profile** (`player.v1.AdminService/GetPlayer`, `ProfilePanel`): signed in or a
  guest, when it was made, its name and whether it is a username or a guest code, the crown, its color, the tiles
  it took, its streak, its messages, its titles (the one worn marked), and every country it plays for and against,
  five of each and the rest behind "All". A guest has all of it too, which the game's public card does not. The
  heading is its flag and its name: the name the profile answers, which is the name it holds now, over the one the
  page that opened it knew; the flag of its row, or else the country it took the most tiles for. `not_found` is an
  account the game keeps nothing about. An address has no profile, and is not asked.
- **An account's page shows what it said, in its conversations** (`chat/TheirMessages`, as a report does): its
  messages by its account id, the chat around each, three conversations at a time. Its lines are marked
  (`line-theirs`), and no line is tagged: there is no report. An address has no messages, since the chat keeps them
  by account, and is not asked.
- **The page inspects on open** (`InspectPlayer`): the antibot's readings, its ban, its pardon and its clicks. The antibot off
  reads as such, not as a failure. A ban or an unban reads it again.
- **The tools**: ban (`BanPlayer`, the ladder's next step by default), unban (`UnbanPlayer`, which forgets the
  latest offence and pardons the key, and says until when; `not_found` is no ban left), revert (`RevertPlayer`, dry run first), and for an account alone
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
- **A flag is an emoji** (`flagOf` in `countries.ts`, `Country`), never the game's flag atlas, which is the
  frontend's: the regional indicators of an ISO code, the tag sequences of England, Scotland and Wales, and nothing
  for Brittany, which Unicode has no flag for. Windows draws the two letters instead, so a flag always goes beside
  its country's name.
- **Replay** (`GetReplay`): a span back from an end (empty: now), saved as `replay-<since>-<until>.json`, the
  answer as protobuf JSON, which the frontend's `npm run clip` reads as it reads what `clips fetch` saves. The
  ledger keeps 72 hours. The file is made in memory (`download.ts`, a `blob:` link), which the CSP allows.

## Clip stats

`src/clips/`, one page (`ClipStatsPage`). **The backend keeps the rows, the page makes the table**, as the home page
counts from the ledger: `clips.v1.AdminService` (the backend's `CLAUDE.md`, "Clip stats") holds a snapshot a day
of each clip on each platform, which the clips job's `clips stats` sends, each row the JSON the job wrote: what
Zernio answered, as it came, and what the clip was.

- **Whether TikTok's business lane is live**, from the last snapshot (`ListSnapshots`): each TikTok account, the type
  TikTok said it was when it was connected to Zernio, how many of its rows hold a number only the business lane gives,
  whether the commercial music library answers (its refusal on hover), and **live** or not: live is the library
  answering or a business number on a post, whatever the type says, since the type is what TikTok said when the
  account was connected to Zernio and goes stale.
- **The snapshots of the last fortnight**: when each was taken, the clips read (and how many not out yet), its rows,
  those still syncing, YouTube's retention curves, and the clips lost that day (why, on hover). A day with no line is
  a day the workflow did not keep.
- **Download CSV** (`tableFileOf` in `actions.ts`): the rows of the last snapshot (each clip once, at its latest age),
  the last 7 or 30 days, or everything, every page of `ListRows` (2000 a page), saved as
  `clip-stats-<since>-<until>.csv` from memory (`saveText`, as the replay is). **`tableOf` (`table.ts`) makes the
  columns from the rows**: the row's own first (`takenAt`, `postId`, `platform`, `ageHours`, …), then one for each
  metric (`metrics.views`, `metrics.impressionSources.forYou`), each part of the metadata (`meta.storyFlag`,
  `meta.record.clip.variants.<experiment>`: an arm is a column with no code) and YouTube's curve (`youtube.points`,
  `youtube.meanRelativeRetentionPerformance`, the mean of its points, and the curve itself). A list, and the two maps
  with a key a country or a second (`audienceCountries`, TikTok's `retentionCurve`), are one cell of JSON. **A metric
  Zernio adds is a column at once**: nothing on the way names a metric. `csvOf` quotes as RFC 4180 says.
- **The numbers are not public**, unlike the rest of the clips job's output: they live behind the admin gate, and
  nowhere in the repository or its workflows.

## Ops

`src/ops/`: `ReconcileTitles` (`TitlesPanel`, after a deploy that adds a title or changes a rule), and `RecountStats`
beside it, which sets each player's bombs and fortified tiles to what planet's ledger holds (once after the deploy
that brought those tracks). Each asks first, and each is harmless twice.

**Game rules** (`RulesPanel`, `rules.ts`) is `planet.v1.AdminService/GetRules` and `ApplyRuleSet` (the backend's
`CLAUDE.md`, "Rules"): the set in force, which is the newest change's or `default` with none, since when and by
whom; every set the config holds, with its refill (`3× as fast`) and its boxes; and the last 50 changes, each with
its request id. **A set is applied from its row**, behind `ConfirmAction`, under a request id of its own
(`crypto.randomUUID()`), so a press sent twice changes the game once, and the page reads the rules again. Seasons
puts the finale's set in force when it starts and `default` when the season ends; a set applied here holds until
the season's next phase. It is how the Final Battle's rules are tried in production by hand, and the backup if the
season's change does not come. Refresh reads them again.

**The charge fills** (`ChargeFills`, `fills.ts`), on the same page, are `GetChargeFills` and `FillCharges` (the
backend's `CLAUDE.md`, "Fills"): the last fill, running or done, and how many accounts it filled; and the last 20,
each with when it started, who started it and its request id. **"Fill every bank"** asks first, then fills every
player's charges to full under a request id of its own (`crypto.randomUUID()`), so a press sent twice fills once.
While a fill runs the page reads them again every 2 seconds (`FOLLOW_MS`), keeping the last answer on screen
(`useLast`, in `useCall.ts`). Seasons fills when the finale starts; this is the backup if that fill does not come.

## Deploy

**A Cloudflare Worker with static assets, `clickplanet-backoffice`** (`wrangler.jsonc`), like the game's. Its
custom domain `backoffice.clickplanet.lol` is set on the Worker in the dashboard, not in `wrangler.jsonc`: a route
there would make the DNS record on the first deploy.

**Workers Builds deploys it from `main`**, on a push that changes something under `apps/backoffice/`: `npm ci && npm
run build`, then `npx wrangler deploy`, with `NODE_VERSION` 22 and `VITE_API_BASE_URL` set on the build. A change
to `proto/` alone ships nothing: the generated code is committed.

`public/_headers` is the page's security:

- **`Content-Security-Policy`**: scripts, styles and images from the page alone, calls to
  `https://api.clickplanet.lol` and the page's own `/assets/` (the tile coordinates) alone, and no frame (`frame-ancestors 'none'`): the pages ban, revert and repaint
  the map, and a framed page could be clicked through. **A new API host is a change here.**
- **The support bucket's host is in `img-src` and `media-src`**, and no other: a note's photos and videos load
  straight from the R2 link `ListNotes` signs (`support.attachments.bucket.endpoint` in `deploy/vps/backend.yaml`).
  **Another bucket is a change here too**, or every attachment shows "not shown here".
- `Referrer-Policy: no-referrer` and `X-Robots-Tag: noindex`.
- `/assets/*` is immutable, `/index.html` is revalidated.

## Layout

```
src/
  api.ts            the transport, one client per AdminService, LONG_CALL_MS
  clients.ts        the narrow client types every page takes, and CLIENTS
  fakeAdmin.ts      FakeAdmin, every admin method for the tests
  refusal.ts        a page's error → signedOut, notOperator, startAgain or failed; an action's in words (wordsOf)
  useCall.ts        one call's state, run on mount, and useLast, its last answer kept while it runs again
  useAction.ts      one read's state, run on a click
  usePage.ts        the page shown, kept in the address bar
  useWide.ts        whether the window is wide enough for the sidebar
  navigation.ts     MENU, the pages by group, and OpenPlayer
  countries.ts      a country id's name and its flag
  App.tsx           the callback, or WhoAmI first, then the page the menu picked
  signIn/           callbackOf, SignInButton, SignInCallback, SignOutButton
  components/       Menu and Sidebar, ConfirmAction, FailureNotice, AccountId and Copyable (with Copy), Country, CountryField,
                    Fact, Person, When
  home/             Home: the tiles, ActivityCharts, Heatmap, CountryTable, PlayingNow, over ledger.ts (the counts),
                    heat.ts (the picture), places.ts (the tile coordinates) and activity.ts
  support/          Ideas and bugs: NoteList, Attachments, labels.ts (names, outcomes, device lines, sizes)
  chat/             Conversation and TheirMessages, which reports/ and players/ show, over chatPages.ts (a page of
                    ListMessages, the parties a report names) and theirConversations.ts
  reports/          Reports: list, detail, the report's own actions
  players/          Players: list, PlayerName, PlayerView, ProfilePanel, the tools on one player, actions.ts
  map/              Reassign a country, Paint random tiles, Download a replay
  ops/              Reconcile titles, Game rules
  gen/grpc/         generated from ../../proto, committed
  gen/map/          the tile coordinates, copied from ../../map, committed
```

**An account is shown by its id, with Copy**, beside its name, wherever a tool takes it: a name can change hands,
and the id is what the operator tools take. **A list of players is the exception**: its rows show the flag and the
name alone, and the id is on the player page each opens.

Plain CSS in `src/index.css`, tokens on `:root`, dark from `prefers-color-scheme`. **It must work on a phone**:
16px gutter, and nothing may scroll sideways, so every id and text wraps (`overflow-wrap: anywhere`).
