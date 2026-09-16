# Production deploy: one DigitalOcean droplet + Cloudflare Pages

Replaces the old DigitalOcean stack (droplet + managed Redis + load balancer +
container registry, ~$50/month). The droplet was never the expensive part — the
managed add-ons were, and none of them are needed. Target cost is **~$6/month**.

| Piece | Where | Cost |
|---|---|---|
| Frontend (static bundle + textures) | Cloudflare Pages | free |
| API (`cmd/api`)             | DigitalOcean droplet, Docker Compose | ~$6/mo |
| TLS | Caddy, automatic Let's Encrypt over DNS-01 | free |
| Images | GitHub Container Registry | free |
| DNS | Cloudflare | free |

Prices are from late 2025 — check before you commit.

## Why DigitalOcean and not something cheaper

Hetzner is roughly 4x the resources for 60% of the price, and its API is good.
But it has no managed Postgres, Redis, Kubernetes or registry: the moment
another project needs a managed data service, that's a new vendor to stack on.
(This app needs none of them — but the next one might.)
DigitalOcean sits at the deliberate middle — enough managed services to grow
into, few enough to stay legible, an API and console that are already familiar.
The delta is about $3/month, which is noise next to the ~$44/month saved by
deleting the add-ons.

Nothing here is DO-specific, though. The stack is Docker Compose plus a
Caddyfile; it moves to any provider that rents a Linux box.

## Layout

- `bootstrap.sh` — provisions a fresh droplet end to end. Run it from your
  laptop with `--host`; it copies itself over and re-runs there as root.
- `docker-compose.yaml` — Caddy + backend, plus a small metrics poller that
  keeps a history the API's in-process counters cannot (see
  [Evidence has to outlive a deploy](#evidence-has-to-outlive-a-deploy-and-by-default-it-does-not)),
  and the postgres that holds the tile map and the chat (see [9. Postgres](#9-postgres)).
- `Caddyfile` — TLS via DNS-01, reverse proxy, CORS
- `caddy/Dockerfile` — Caddy built with `caddy-dns/cloudflare`. The stock image
  has no DNS provider module and cannot solve the DNS-01 challenge.
- `backend.yaml` — API config; secrets come from env, not this file
- `.env.example` — copy to `.env` on the box (`bootstrap.sh` writes it for you)

`deploy/docker-compose.yaml` (one level up) stays as the *local* full-stack
compose. This directory is only for the production box.

## 1. The Cloudflare API token

Caddy issues its TLS certificate with the ACME **DNS-01** challenge: it proves
control of the domain by writing a `_acme-challenge` TXT record through the
Cloudflare API, rather than by answering an inbound request on port 80. That is
what lets the origin stay hidden — the A record can be proxied and port 80 never
has to be reachable from the public internet, at renewal as well as at first
issuance.

Create the token at **dash.cloudflare.com → My Profile → API Tokens → Create
Token → Create Custom Token**.

### Permissions

Exactly two, both required:

| Type | Resource | Level | Why |
|---|---|---|---|
| Zone | **DNS** | **Edit** | Create and delete the `_acme-challenge` TXT record |
| Zone | **Zone** | **Read** | Look up the zone ID that owns the domain |

`Zone / Zone / Read` is easy to miss and the failure is opaque — without it the
plugin cannot resolve which zone `api.clickplanet.lol` belongs to, and issuance
fails before any TXT record is attempted.

### Zone resources

| Field | Value |
|---|---|
| Include | **Specific zone** → `clickplanet.lol` |

Scope it to the single zone. "All zones" grants DNS edit over every domain on
the account, and this token lives in a file on a $6 box.

### The rest

- **Client IP filtering** — optional, and if you use it the allowed address is
  the **droplet's**, not your laptop's. Caddy is the only thing that ever uses
  this token and it runs on the box, so a filter set from the machine where you
  created the token fails with `[9109] Cannot use the access token from
  location: <droplet ip>` and nothing else. Pinning it to the droplet is good
  hardening — a leaked token is then useless anywhere else — but remember to
  update it if you rebuild the box on a new address, and note that you will no
  longer be able to test the token from your laptop.
- **TTL / expiry** — leave it unset, or set a calendar reminder. Certificates
  renew roughly every 60 days with no human involved; an expired token turns
  that into a silent failure that surfaces as an outage two months later.

Copy the token when it is shown — Cloudflare will not display it again. It goes
to `bootstrap.sh --cf-token`, which writes it to `.env` (mode 600) on the box as
`CLOUDFLARE_API_TOKEN`. It is the only secret in this stack.

To check it works before deploying:

```bash
curl -sS -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" \
  https://api.cloudflare.com/client/v4/user/tokens/verify
```

`"status": "active"` means the token is valid — though that endpoint does not
confirm the two permissions above, only that the token exists.

## 2. DNS

Both records on `clickplanet.lol` are **proxied**:

| Name | Type | Value | Proxy |
|---|---|---|---|
| `api` | A | Droplet IPv4 | **Proxied (orange cloud)** |
| `@` / `www` | CNAME | Pages target | Proxied (orange) |

The orange cloud on `api` is load-bearing here, not cosmetic. `bootstrap.sh`
restricts ports 80 and 443 to Cloudflare's published IP ranges, so a grey-clouded
record means visitors reach the box directly and get refused by ufw — and it
publishes the origin IP into DNS, which is the thing this setup avoids.

Set the zone's SSL/TLS mode to **Full (strict)**. Caddy presents a real
Let's Encrypt certificate, so strict verification passes; the Flexible mode
talks plain HTTP to the origin, which Caddy redirects to HTTPS, giving a
redirect loop.

Long-lived server-streaming responses pass through the proxy on the free plan, provided they are never silent — see `httpServer.streamHeartbeat`.

## 3. The droplet

A **Basic / Regular $6 droplet** (1 vCPU, 1 GB RAM, 25 GB SSD, 1 TB transfer) is
enough: the Go API holds the whole tile grid in a few MB and there is no
database beside it. Pick the Ubuntu LTS image and a region near your players,
and attach your SSH key at creation.

That 1 GB is enough at rest but has no swap by default, so a transient spike —
an apt upgrade, docker unpacking layers — is an OOM kill rather than a slow
moment, and the killer picks the biggest process, which is the API holding the
game state. `bootstrap.sh` adds a 2 GB swap file with `vm.swappiness=10` (an
emergency buffer, not somewhere to page the working set); `--swap none` skips
it.

`apps/backend/Dockerfile` pins `GOARCH=amd64`, so stay on a **regular Intel/AMD**
droplet — the backend image will not run on an ARM one, and `bootstrap.sh`
refuses to continue if it finds itself on `aarch64`.

Everything else is one command from your laptop:

```bash
./deploy/vps/bootstrap.sh \
  --host YOUR_DROPLET_IP \
  --api-domain api.clickplanet.lol \
  --frontend-origin https://clickplanet.lol \
  --cf-token 'YOUR_CLOUDFLARE_TOKEN'
```

It copies itself to the box over SSH and re-runs there as root, then: installs
Docker and turns off its userland proxy (see below), creates the `deploy` user, generates and installs a CI keypair
(`~/.ssh/clickplanet_ci`, private half never leaves your laptop), restricts ufw
to SSH plus Cloudflare's ranges on 80/443, clones the repo to
`/opt/clickplanet`, writes `.env`, builds
Caddy with the Cloudflare DNS plugin, and starts the stack. It finishes by
printing the `gh secret set` commands for step 5.

It is idempotent — re-running after a failure skips whatever is already done —
and it stops with a specific message rather than a confusing one when something
is not ready: the wrong CPU architecture, a token you did not pass, a
grey-clouded DNS record, or a backend image that is not pullable yet.

On first boot the API finds no tiles in postgres and starts from an empty map,
logging `loaded the tile map` with `ownedTiles=0`. Check it:

```bash
curl -sS 'https://api.clickplanet.lol/planet.v1.ClickService/MapDensity?connect=v1&encoding=json&message=%7B%7D'
```

If the certificate does not appear within a few minutes, the DNS-01 challenge is
where to look:

```bash
ssh deploy@YOUR_IP 'cd /opt/clickplanet/deploy/vps && docker compose logs caddy | grep -i -e acme -e cloudflare -e certificate'
```

An `unauthorized` or zone-lookup error there almost always means the token is
missing one of the two permissions in step 1.

### Docker's userland proxy is off, and must stay off

With it on, every request on some Cloudflare connections reaches the API as the
same caller, `172.18.0.1`. `docker-proxy` starts listening on 80/443 slightly
before the NAT rule that forwards to Caddy exists. A connection that lands in
that gap stays on `docker-proxy` for its whole life, and Caddy sees the bridge
gateway as its peer. That is not a Cloudflare range, so Caddy ignores
`Cf-Connecting-Ip`. Cloudflare keeps origin connections open for hours and
shares them between visitors. On 2026-09-14 one connection, opened 1.8 s after a
Caddy restart, carried 78% of all clicks. It merged a bot into a crowd, so no
watchdog could see it, and it put the crowd one metronome flag away from a
persistent ban. Every deploy restarts Caddy.

`bootstrap.sh` sets `"userland-proxy": false` in `/etc/docker/daemon.json` and
restarts Docker once when it changes. Re-running it on an existing box applies
the fix, with a few seconds of downtime. To check a running box:

```bash
ssh root@YOUR_IP 'docker info | grep EnableUserlandProxy; ss -tn src 172.18.0.1 dport = :443'
```

`false` and an empty socket list means every request keeps its real address.

## 4. Frontend on Cloudflare Pages

Create a Pages project from the GitHub repo:

| Setting | Value |
|---|---|
| Root directory | `apps/frontend` |
| Build command | `npm ci && npm run build` |
| Output directory | `dist` |
| Env var | `VITE_API_BASE_URL=https://api.clickplanet.lol` |
| Env var | `VITE_TURNSTILE_SITEKEY=0x4AAAAAAEuFCQwAVFUYug8z` (see [§5](#5-turnstile-and-the-click-session)) |

Generated protobuf code is committed, so the build needs no `buf`.

`npm run build` also copies `static/` into `dist/static/` and deletes the 24 MB
`coordinates.json` from it — that file is the human-readable generator output,
kept in the repo but never served. The viewer fetches
`static/coordinates-<hash>.bin` at runtime instead (~5 MB, no JSON parse), so
the coordinates are no longer in the JS bundle and Cloudflare's 25 MiB
per-file limit is no longer close. `copy:static` fails the build if that
`.bin` is missing — run `npm run coordinates:convert` if you hit that.

The `.bin` name carries a content hash, and `public/_headers` caches it
`immutable` for a year: regenerating the tile map ships a new file name, so a
stale blob can never be served. If you regenerate it, `gameMap.maxIndex` in
`backend.yaml` must be updated to match the new tile count.

## 5. Turnstile and the click session

The API refuses a `Click` that carries no session token it minted, and the only
way to get one is to pass Turnstile. Without this configured on both sides the
game still runs — `auth.enabled: false` keeps the old address-only
behaviour — but the anti-bot floor is back to what a blocklist can do.

### The widget

Already created, as **clickplanet click session**:

| | |
|---|---|
| Sitekey | `0x4AAAAAAEuFCQwAVFUYug8z` — public, goes in the Pages build |
| Secret key | dash.cloudflare.com → Turnstile → the widget → *Settings*; goes in `.env` only |
| Hostnames | `clickplanet.lol`, `www.clickplanet.lol` |
| Mode | Managed |
| Pre-clearance | none |

To recreate it, or to make a separate one for development:
dash.cloudflare.com → **Turnstile** → *Add widget manually* (**not** *Configure
with Spin* — that path rewrites the frontend and backend integration, which this
repo already has):

| Setting | Value |
|---|---|
| Domains | `clickplanet.lol`, `www.clickplanet.lol` |
| Widget mode | Managed |

**Do not add `localhost`.** The widget's domain list and the backend's
`auth.turnstile.hostnames` are checked against the hostname siteverify
reports, and a production allowlist that admits localhost admits a token minted
from a page an attacker controls locally. Use a second, separate widget for
development if you want one.

It gives you two keys:

- the **sitekey**, public — it is read straight off the page — which goes in the
  Pages build as `VITE_TURNSTILE_SITEKEY`
- the **secret key**, which goes in `.env` on the droplet as `TURNSTILE_SECRET`
  and never leaves it

### Fill in `.env`

`bootstrap.sh` generates `SESSION_SECRET` and leaves `TURNSTILE_SECRET` empty.
On an existing box, set the Turnstile one by hand — copy it from the dashboard
rather than reading it out anywhere it could be logged:

```bash
# On the droplet, in the stack directory.
read -rsp 'Turnstile secret: ' s && echo && printf 'TURNSTILE_SECRET=%s\n' "$s" >> .env && unset s
```

To check a secret is the right one before trusting it, without a browser:

```bash
read -rsp 'Turnstile secret: ' s && echo && curl -s \
  https://challenges.cloudflare.com/turnstile/v0/siteverify \
  -d "secret=$s" -d 'response=dummy'; unset s; echo
```

`"error-codes":["invalid-input-response"]` means the **secret is valid** and only
the dummy token was rejected. `invalid-input-secret` means it is not.

If you ever need to regenerate `SESSION_SECRET`:

```bash
openssl rand -hex 32
```

`SESSION_SECRET` signs the tokens; anyone holding it can mint one the API will
accept. **It cannot be left empty** while `auth.enabled` is true: the mint and
the click check each derive their signer from it, so a server that invented one
would invent a different one per context and could not verify what it had just
minted. The stack refuses to start instead, naming the variable. Rotating it
deliberately is cheap — every client mints again on its next click.

`.env` is gitignored. `.env.example` beside it is the template and is committed.

### Roll it out in three steps

`backend.yaml` ships `auth.enabled: false`, and `auth.enforce: false`
under it. Nothing below breaks a running site at any point: the API starts
minting before anything requires a session, and starts requiring one only once
the clients that can mint are the overwhelming majority.

1. Set `auth.enabled: true` in `backend.yaml` (leave `enforce` false) and
   deploy the backend. Then watch:

   ```bash
   docker compose exec backend wget -qO- localhost:8080/metrics | grep click_session_checks
   ```

   `verdict="missing"` is every click from a client that sends no token —
   at this point, all of them.

2. Deploy the frontend with `VITE_TURNSTILE_SITEKEY` set. `verdict="valid"`
   should climb and `missing` should fall away as caches expire.

3. Once `valid` is the overwhelming majority, set `auth.enforce: true` in
   `backend.yaml` and redeploy.

Flipping `enforce` before step 2 has settled refuses real players with a 401.
The number to watch is `missing`, not the clock.

### If clicks start failing with 401

- **Every click, right after enabling** — the frontend build has no sitekey, or
  Pages was not rebuilt after the env var was added.
- **Every click, in the browser only** — check the preflight. `Caddyfile` must
  list `X-Session-Token` in `Access-Control-Allow-Headers`; a custom header on a
  cross-origin POST is what makes it preflighted, and a preflight that omits it
  fails the click before the API ever sees it.
- **Only some players** — an ad blocker or privacy extension blocking
  `challenges.cloudflare.com`. That is what `SessionUnavailableModal` tells them
  to fix; there is nothing to do server-side.
- **`CreateSession` itself answering 403** — the API log says which check
  tripped (the caller is never told). Usually `hostnames` not matching the
  origin the widget is actually embedded on.


## 6. Watching for bots

Sessions raise the floor to "drive a real browser". What gets through that is a
userscript in a real browser, holding a genuine session — and the only thing
left that separates it from a player is behaviour.

`antiBot` watches seven behaviours, one per watchdog:

- **`retaker`** — takes a tile back moments after losing it, over and over, in a
  band no hand holds.
- **`sequencer`** — walks the tile ids rather than the map: 1, 2, 3, 4, on and on
  until a continent is painted.
- **`metronome`** — never varies and never stops. Timed between clicks *tried*,
  429s included, so the throttle cannot hide a steady loop.
- **`defender`** — nearly every take wins back a tile its country just lost.
  **Measuring only**: it sets no verdict until `minShare`/`certainShare` are set.
- **`catcher`** — catches every bonus box, at once.
- **`cohort`** — starts, paces and stops in step with other scopes, group after group.
- **`scraper`** — reads the whole map again and again. The web app reads it once
  per page load and opens one stream with it; a map read beyond one per stream
  is a client that is not the web app.

Each returns `certain` or `suspect`. **`certain` bans on its own; `suspect` is a
reading that would ban real players if it were trusted alone**, and counts only
alongside a second watchdog. `jury.minSuspects` (2) is how many it takes.

A flagged caller's clicks are answered `OK` and dropped. That is deliberately not
a refusal — a 403 names the check that tripped, and a silent no-op names nothing,
so working around it is guesswork instead of a diff. It is not permanent: the
caller reads the map back over the same stream and will notice eventually.

`backend.yaml` now runs with `shadowBan.enforce: true`. It spent 2026-09-11 in
observe mode (`enforce: false`, which judges and logs without dropping anything)
and the bounds below were set from what that pass recorded — see "What the
measuring pass found". **Anything you widen here, widen back in observe mode
first.**

### The histogram says where the line is

```bash
docker compose exec backend wget -qO- localhost:8080/metrics | grep click_reaction
```

Every click that takes a tile another caller just took is timed into
`click_reaction_seconds`. Read the bucket counts directly: a reflex bot driven
by the update stream piles up under 150 ms, while a bot on a timer sits in a
narrow band wherever its timer is — one seen in production answered at almost
exactly 1 s. Human reactions spread out broadly and do neither.

The buckets step evenly to 2 s for that reason. An earlier set jumped 0.5 → 1 →
2, and a ~1 s bot was invisible in it: every reaction landed in two enormous
buckets that could not tell a tight timer from a broad human.

**The histogram is global.** It mixes the bot, the players reacting to the bot,
and everyone else, so it tells you *that* there is a band and roughly where —
never which caller owns it. It also only sees `retaker`; the other two watchdogs
have no histogram, because a sweep has no delay to time. Per-caller numbers come
from the log below.

### Setting the defender's shares

```bash
docker compose exec backend wget -qO- localhost:8080/metrics | grep click_retake_share
```

Once a minute, every caller with `minClicks` takes in the last `trackWindow` adds
its retake share to `click_retake_share`. The poller keeps it. A defence loop sits
at the top bucket for as long as it runs; a painter sits low. **Two people fighting
over one tile also sit at the top**, for as long as the fight lasts — so read how
long callers stay there, not only that they get there, and set `certainClicks`
past what a fight lasts.

### The log says who

The address is never a metric label — that is unbounded cardinality, and it
would put personal data in every scrape. It goes to the log instead:

```bash
docker compose logs backend | grep "antibot ban"
```

```
level=WARN msg="antibot ban" scope=198.51.100.20 flags=2 clicks=41022
  activeFor=11h20m2s longestGap=1.9s topCountry=fr topCountryClicks=41022
  tiles="[184430 184431 184432 ...]"
  retaker="clear"
  sequencer="certain stride share=0.991 steps=200 stride=1"
  metronome="certain cadence clicks=900 median=1.002s spread=11ms sustained=6h2m"
```

**Every watchdog is on the line, including the ones that said `clear`.** What did
not fire is half of reading a ban that did. The line above is the overnight
sweep: it never fought anyone for a tile, so `retaker` has nothing to say about
it, and under the old single-behaviour rule it would have run all night unseen.

**Read `spread`, not `median`.** A caller clicking at almost exactly one second,
nine hundred times, within 11 ms of itself, is running a timer — the delay is
human-looking on purpose and only the regularity gives it away. The same is true
of `retaker`: its `maxSpread` is the bound that does the work.

**`share` is `sequencer`'s equivalent.** 0.991 of two hundred steps sitting at
`stride=1` is a loop over an integer. Tile ids follow the icosahedron's vertex
order, so a hand filling in a shape does not produce that, whatever shape it is
filling.

**`flags` is how many times this caller has crossed the bar.** At
`reflagInterval` ≥ the watchdog's `trackWindow`, each flag rests on evidence the
previous one never saw, so `flags=6` is six independent judgements agreeing. A
caller that flags once and never again was probably a bad five minutes; one whose
count keeps climbing is a standing pattern.

**`activeFor` and `longestGap` are the persistence signal, and the one a
randomised delay cannot beat.** A bot author who reads this can jitter the delay
until `spread` looks human — and `metronome` will then say `clear`, which is
exactly why there is more than one watchdog. What costs them something real is
stopping. Eleven hours of `activeFor` with `longestGap` under two seconds is
nobody's evening. Neither number feeds a rule on its own, because deciding on
them would ban the genuinely obsessed.

`topCountry` is the country the caller painted with most. It is **context, not
evidence** — the client declares it in the request, so it is changed by editing
one string, and plenty of real players paint the same flags a bot does. Read it
to understand what a caller was doing; never widen a rule to act on it.

Before loosening any of these bounds, get into a tile war yourself and confirm
your own line's numbers sit clearly outside the ones you are about to set.

### What the measuring pass found

The observe pass on 2026-09-11 flagged six times, all one caller on one IPv6
/64. Reading those six lines changed two of the three watchdogs' bounds.

**`sequencer` was the only one whose evidence held, and its shipped bounds were
already right.** The caller tracked tile id N+1 for 150 of 200 steps. Walking
the ids is walking a geodesic: the id path turns **0.2° per step**, so following
it draws a ruler-straight line across the map, and with ~6 neighbours per tile a
hand choosing freely holds that for a few clicks rather than a hundred and fifty.
Simulating a perfectly straight human sweep — better than a person manages, one
click at a time with no drag to help — lands on N+1 a median **0.000** of the
time. Forty steps at a constant stride is past anything a hand produces; two
hundred is not arguable.

Note what this is *not*: N+1 being spatially adjacent to N proves nothing on its
own, and the ids are laid out so it nearly always is. The signal is holding one
stride, not the stride being small.

**`retaker.maxMedian: 2s` was letting the retaker ban alone, and four of the six
flags were that.** A median above `maxMedian` only demotes `certain` to
`suspect` — and 2s is slower than any human reaction, so it could never fail.
Every retaker suspicion arrived as `certain`, `certain` bans without a second
watchdog, and `jury.minSuspects` was never consulted. The four lines it produced
were humans in a tile war, at spreads of 689–879ms. Now `maxSpread: 300ms` /
`maxMedian: 250ms`, which sits between those humans and the 138ms bot above.

**`metronome.maxSpread` stays at 400ms on purpose.** The caller read 248ms, but
a player spam-clicking into the 1 click/s throttle has its surviving clicks
handed back at the refill rate — the rate limiter runs *before* the antibot
interceptor, so this number partly measures the limiter. That is tolerable at
`suspect`, which needs a second watchdog to mean anything. `certainFor: 30m` and
`certainClicks: 900` are the ones that ban alone here, and the claim they rest on
is the absence of any pause past `maxGap`, not the spread.

Then `shadowBan.enforce: true` and redeploy.

`shadowban_flagged` is how many callers are inside a ban and **counts while
`enforce` is false too** — a non-zero gauge in observe mode means the rules are
biting, not that anything was dropped. `shadowbanned_clicks` is the one that
stays at 0 until you enforce.

`shadowban_flags` counts flags rather than callers, **labelled by watchdog**, so
the two read together: `shadowban_flags{watchdog="sequencer"} 40` against
`shadowban_flagged 2` is two callers flagged twenty times each, which is a very
different picture from forty callers caught once. The labels also tell you which
watchdog is earning its keep before you enforce. All of them are readable with
the `wget` line above.

**A ban is the only thing those two count, so a day with no ban reads as
nothing.** On 2026-09-14 a bot attack produced zero bans and no sign of how
close the watchdogs came. Two series fill that gap, both labelled
`{watchdog, level}` with `level` `suspect` or `certain`:

```bash
docker compose exec backend wget -qO- localhost:8080/metrics | grep antibot_opinions
```

- `antibot_opinions_total` counts **rises**: a watchdog's reading of a caller
  reaching a level it has not held within `jury.suspicionWindow` (10m). A
  reading flapping across a bound counts once a window, not once a click; one
  that lapses and comes back counts again.
- `antibot_opinions_standing` is how many callers each watchdog reads at that
  level right now, set once a minute by the jury's sweep.

**Levels are cumulative**: `suspect` includes every `certain`, so
`suspect − certain` is the near misses. `antibot_opinions_total{watchdog="sequencer",level="suspect"} 30`
with `shadowban_flags` still at 0 is thirty suspicions nobody corroborated — look
at what the other watchdogs were reading on the same callers before loosening
`jury.minSuspects`. The poller keeps both.

Bans escalate: 24h for a first offence, 7 days for a second, 3 years from the
third. A caller that keeps going while banned only extends the ban it has. Bans
are kept in postgres, in `antibot.bans`, so a deploy keeps them. What the
watchdogs are tracking is kept beside them, in `antibot.evidence`, so a restart
does not start their windows again; it keeps three days at most. Both are
written every minute and once more on a clean shutdown.

See every running ban:

```bash
docker compose exec postgres psql -U clickplanet -c "select * from antibot.bans where banned_until > now() order by banned_until"
```

Unban one scope. Stop the backend first: the running one holds its bans in
memory, keeps the ban running and writes it back.

```bash
docker compose stop backend
docker compose exec postgres psql -U clickplanet -c "delete from antibot.bans where scope = '1.2.3.4'"
docker compose start backend
```

That forgets its offences too. To end the ban and keep them, so its next ban
climbs the ladder: `update antibot.bans set banned_until = now() where scope = '1.2.3.4'`.

Set `enforce` back to false to stop dropping clicks for everyone at once.
### Evidence has to outlive a deploy, and by default it does not

Everything above is in-process. A deploy pulls a new image and **recreates** the
container, which resets every counter and histogram to zero — and, less
obviously, deletes the log too: Docker's default `json-file` driver stores logs
per container id, so the old container taking its `antibot ban` lines
with it is the bigger loss of the two. Two pieces of the compose file exist for
this.

**The backend logs to journald**, which is stored on the host rather than under
the container:

```bash
journalctl CONTAINER_NAME=cp-backend --since '2 days ago' | grep "antibot ban"
```

`docker compose logs backend` still works and still shows only the current
container, which is usually what you want; `journalctl` is how you read across a
deploy. This needs a persistent journal — check `journalctl --disk-usage`, and
if `/var/log/journal` does not exist the journal is memory-only and this buys
nothing.

**`cp-metrics-poller` scrapes `/metrics` every `POLL_INTERVAL` seconds** and
appends the shadowban and click series to the `metrics_history` volume, one
file per UTC day, pruned after `POLL_RETENTION_DAYS`:

```bash
docker compose exec -T metrics-poller ls /history
docker compose exec -T metrics-poller cat /history/metrics-2026-09-11.prom
```

Each block is stamped with the time it was taken. The values are cumulative
since the API started, so **you read this by subtracting two blocks**, not by
looking at one:

```bash
docker compose exec -T metrics-poller sh -c \
  "grep -A22 '^# 2026-09-11T06' /history/metrics-2026-09-11.prom | head -23"
```

A block whose numbers are *lower* than the block above it is where a deploy
happened and the counters restarted. Subtract within a run, never across one.

It is not a Prometheus, deliberately: a real one is 80–150 MB resident beside a
1 GB droplet already holding the tile map, and the job here is comparing two
samples a day apart. If this ever needs `histogram_quantile` and proper
reset-aware `rate()`, that is the moment to spend the memory — the poller is
then deleted, not extended.

### Reading the access log

On 2026-09-14 a bot attack came through Cloudflare VPN addresses. Then `cp-caddy`
was recreated, and `docker logs` kept 16 lines. There was no record of who sent
what. So Caddy now writes every request as one JSON line to
`/var/log/caddy/access.log` on the `caddy_logs` volume. A recreated container
keeps it.

- Caddy rolls the file at 100 MiB and gzips the old one. It deletes rolls older
  than **14 days**, or past 150 rolls (about 1.5 GB) if an attack writes more.
- The same lines still go to `docker logs cp-caddy`. That copy is lost on
  recreate.
- **The access log holds personal data**: client IPs, user agents, countries.
  Like the chat messages, 14 days is a policy decision. Shorten `roll_keep_for` in the
  `Caddyfile` to hold less. Nothing backs up this volume.
- `X-Session-Token` is written as `REDACTED`. It is a bearer token. You can see
  if a request had one, not what it was.

Useful fields:

| Field | What |
|---|---|
| `ts` | Unix seconds |
| `request.client_ip` | The visitor, from `Cf-Connecting-Ip` (trusted only from Cloudflare ranges) |
| `request.uri` | `/planet.v1.ClickService/Click`, etc. |
| `status`, `duration` | HTTP status, seconds |
| `request.headers["User-Agent"][0]` | User agent |
| `request.headers["Cf-Ray"][0]` | Cloudflare request id |
| `request.headers["Cf-Ipcountry"][0]` | Country Cloudflare placed the visitor in |

Run these on the box, in the stack directory (`bootstrap.sh` installs `jq`).
Every command starts with the same line, which reads the old rolls then the
current file:

```bash
docker compose exec -T caddy sh -c 'zcat /var/log/caddy/*.gz 2>/dev/null; cat /var/log/caddy/access.log' > /tmp/access.jsonl
```

Top callers by `Click` count:

```bash
jq -r 'select(.request.uri == "/planet.v1.ClickService/Click") | .request.client_ip' /tmp/access.jsonl | sort | uniq -c | sort -rn | head -20
```

Status by path (query string cut off):

```bash
jq -r '"\(.status) \(.request.uri | sub("\\?.*"; ""))"' /tmp/access.jsonl | sort | uniq -c | sort -rn
```

Every request of one scope in a time range (UTC). For an IPv6 /64, change
`.request.client_ip == $ip` to `(.request.client_ip | startswith($ip))` and give
the prefix, `2001:db8:1:2:`:

```bash
jq -c --arg ip 203.0.113.7 --arg from 2026-09-14T06:00:00Z --arg to 2026-09-14T09:00:00Z 'select(.request.client_ip == $ip and .ts >= ($from | fromdate) and .ts < ($to | fromdate)) | {time: (.ts | todate), status, uri: .request.uri, ua: .request.headers["User-Agent"][0], ray: .request.headers["Cf-Ray"][0], country: .request.headers["Cf-Ipcountry"][0]}' /tmp/access.jsonl
```

`/tmp/access.jsonl` is a copy of the personal data. Delete it when you are done.

## 7. CI and the image registry

`.github/workflows/deploy-backend.yml` builds the image to GHCR and rolls the
container over SSH. GHCR rather than a dedicated registry because it adds no
account, no vendor and no bill: the code is already on GitHub, CI already runs
there, and the push authenticates with the built-in `GITHUB_TOKEN`.

Repository secrets required:

- `VPS_HOST` — the droplet's IP
- `VPS_USER` — `deploy`
- `VPS_SSH_KEY` — private key whose public half is in `deploy`'s `authorized_keys`

`bootstrap.sh` generates that keypair at `~/.ssh/clickplanet_ci`, installs the
public half on the box, and prints the exact commands to set all three:

```bash
gh secret set VPS_HOST    --body 'YOUR_DROPLET_IP'
gh secret set VPS_USER    --body 'deploy'
gh secret set VPS_SSH_KEY < ~/.ssh/clickplanet_ci
```

The workflow builds **two** images: the backend, and Caddy with the
`caddy-dns/cloudflare` plugin from `caddy/Dockerfile`. Both go to GHCR and the
droplet only ever pulls — linking Caddy with that plugin pulls in certmagic,
quic-go, smallstep, otel and the AWS SDK, which is minutes of CPU and enough
memory to OOM a 1 GB box mid-deploy.

Both GHCR packages must be set to **Public** after their first build, not just
the backend one.

The `Caddyfile` is read from the checkout at container start, so editing it
needs only `docker compose up -d caddy` (or a `bootstrap.sh` run), no rebuild.
Changing `caddy/Dockerfile` means waiting for CI to publish a new image.

**One-time after the first successful build:** a new GHCR package is created
private even when the repo is public. Open
`https://github.com/users/raphoester/packages/container/clickplanet-backend/settings`,
set visibility to **Public**, and link it to the repo. The droplet can then
`docker pull` anonymously — no registry credentials on the box at all. If you'd
rather keep the package private, generate a read-only PAT with `read:packages`
and run `docker login ghcr.io` once as `deploy`.

Pages deploys itself on push; no workflow needed.

## 8. Live chat

Chat is always on. The API serves `/chat.v1.ChatService/`, `ListenForEvents`
included, and Caddy already forwards it. `SendMessage` is an
**unauthenticated public write endpoint** — anyone who can reach the API can
post, under any name — so the things that keep it usable are all config:

| Knob | Where | Default here |
|---|---|---|
| Per-IP throttle | `chat.rateLimiter` | one message per 3s, 5 in hand |
| Cutting someone off | `chat.blockedIPs` | CIDRs, `203.0.113.7/32` for one address |
| Message log retention | `chat.storage.retention` | 30 days |
| Sender tag salt | `CHAT_TAG_SALT` in `.env` | generated by `bootstrap.sh` |

**The salt is the one secret this service takes**, which is why it is in `.env`
rather than in `backend.yaml`. A sender's visible tag is a hash of their address
and this salt; publishing the salt would let anyone confirm a guessed address
against a tag. `bootstrap.sh` writes one on a fresh box and adds one to an
existing `.env` that has none — it never overwrites a salt already there,
because **changing it renames every sender at once**. To set or rotate it by
hand:

```bash
ssh deploy@YOUR_IP
cd /opt/clickplanet/deploy/vps
sed -i "/^CHAT_TAG_SALT=/d" .env
echo "CHAT_TAG_SALT=$(openssl rand -hex 16)" >> .env
docker compose --env-file .env up -d backend
```

Leaving it empty is not fatal but is worse than any fixed value: the API
generates a fresh salt at every boot, logs `no chat.service.tagSalt configured`,
and every tag changes on each restart.

**The chat messages hold personal data.** One row per message in `chat.messages`,
with the sender's IP beside their text. `chat.storage.retention` (30 days) is a
policy decision, not a cache size: an hourly prune deletes older rows. Shorten
it if you would rather hold less. A message that cannot be written to postgres
is refused, not broadcast: the table is the audit trail.

Anything in `backend.yaml` can also be overridden from the `environment:` block
instead — `cfgutil` reads env vars with `.` as the nesting delimiter, so the key
is the config path verbatim, which is how `CHAT_TAG_SALT` reaches
`chat.service.tagSalt`.

## 9. Postgres

The tile map, the ledger and the chat are kept in the `postgres` service, on the
`pg_data` volume, and so are the antibot's bans and evidence. The API loads them
at boot, writes what changed every second (bans and evidence every minute), and
once more on a clean shutdown; each chat message is written before it is
broadcast. It is not published on any port: only the backend
reaches it. Each backend module keeps its tables in a schema of its own (`planet`
for the tile map and the ledger, `antibot` for bans and evidence, `chat` for the
messages) and migrates it at boot. The API refuses to start without postgres.

**The password is `POSTGRES_PASSWORD` in `.env`.** `bootstrap.sh` generates it.
Without it, `docker compose up` refuses to start. Never change it: postgres reads it only when `pg_data` is empty, so a
new value locks the API out of the existing data.

How many tiles and takes it holds:

```bash
docker compose exec postgres psql -U clickplanet -c "select count(*) from planet.tiles"
docker compose exec postgres psql -U clickplanet -c "select count(*) from planet.ledger_takes"
```

A psql shell: `docker compose exec postgres psql -U clickplanet`.

### Backups

**Nothing is backed up yet.** All state is in postgres. For a copy by hand:

```bash
docker compose exec postgres pg_dump -U clickplanet -n planet clickplanet > planet-$(date +%F).sql
docker compose exec postgres pg_dump -U clickplanet -n antibot clickplanet > antibot-$(date +%F).sql
docker compose exec postgres pg_dump -U clickplanet -n chat clickplanet > chat-$(date +%F).sql
```

DigitalOcean's droplet backups (+20% of the droplet price, so ~$1.20/mo) cover
the whole disk if you would rather not think about it.

A droplet set up before 2026-09-15 still has the `vps_tile_state` volume, with
the `.imported` files of the first postgres boot, and a nightly cron that tars
it. Nothing reads them. `chat.log.imported` holds personal data. To remove both,
as `deploy`, once the backend runs without the mount:

```bash
crontab -l | grep -v 'vps_tile_state\|tiles-\*' | crontab -
docker volume rm vps_tile_state
rm -f ~/backups/tiles-*.tar.gz
```

## 10. Operator tools

`httpServer.adminBindAddress` serves the backend's operator services
(`planet.v1.AdminService`) on `127.0.0.1:8081`, inside the container. They are
not behind Caddy and have **no authentication**: loopback is their whole
protection, so a non-loopback address refuses the boot. Reach them from the box
with `docker compose exec`. They are ordinary Connect RPCs, so a request is a
JSON POST to `/<package>.<Service>/<Method>`.

### Give one country's tiles to another

Dry run first. It changes nothing and says how many tiles each side holds:

```bash
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{"fromCountryId":"dz","toCountryId":"fr","dryRun":true}' http://127.0.0.1:8081/planet.v1.AdminService/ReassignCountry
```

Then the same without `"dryRun":true`. There is no restart:

- It moves every tile `from` holds, 256 at a time every 50ms — about 4.5s for
  22,000 tiles.
- Each tile goes out on the live stream as an ordinary update, so open tabs
  repaint, the toll sees the new counts, and the next flush writes it to postgres.
- A tile `from` takes back while it runs stays theirs. `fromAfter` in the answer
  says how many; run it again.
- **A count of zero is left out of the answer** — that is how protobuf JSON
  writes it. `{"fromBefore":22040,"moved":22040,"toAfter":22040}` means `fromAfter` is 0.
- A refusal (unknown or identical country) shows as `server returned error: HTTP/1.1 400`.
- Every call is logged: `journalctl CONTAINER_NAME=cp-backend | grep "admin country reassignment"`.

**Reassigning back does not undo it**: it would also move the tiles `to` held
before. Copy the table first if you may want to return:

```bash
docker compose exec postgres psql -U clickplanet -c "create table planet.tiles_before_reassign as table planet.tiles"
```

To go back: stop the backend (its last flush runs on the way down), then
`truncate planet.tiles; insert into planet.tiles select * from planet.tiles_before_reassign;` in psql,
then start it.

### Paint random tiles of a country with a flag

Paints `count` tiles with `flagCountryId`, starting on `areaCountryId`'s ground.
Leave out `areaCountryId` to start anywhere on the map.
Dry run first; it says how many tiles of the area do not wear the flag yet:

```bash
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{"flagCountryId":"dz","areaCountryId":"fr","count":500,"proximity":0.8,"dryRun":true}' http://127.0.0.1:8081/planet.v1.AdminService/PaintRandomTiles
```

- `proximity` goes from 0 to 1. 0 scatters the tiles over the whole country;
  1 grows one patch. Between the two you get a few patches.
- A patch can grow past the country's border. `outsideArea` says how many
  tiles it took there.
- A tile somebody takes while it runs stays theirs: `painted` can be below `picked`.
- It is not undone by anything. Copy the table first, as for a reassign.
- Every call is logged: `journalctl CONTAINER_NAME=cp-backend | grep "admin random paint"`.

### Find, ban and revert one player

For a pattern you see on the map and no watchdog catches. A player is a
**scope**: the address over IPv4, the /64 over IPv6.

Who painted the `ps` flag on Israel's ground, held or painted over since,
latest take first (`limit` is 20 when left out; leave out `areaCountryId` for
the whole map):

```bash
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{"flagCountryId":"ps","areaCountryId":"il"}' http://127.0.0.1:8081/planet.v1.AdminService/FindPlayers
```

Who took the most tiles, over every flag and the whole map: most takes first,
then most tiles held (`limit` is 20 when left out):

```bash
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{}' http://127.0.0.1:8081/planet.v1.AdminService/TopPlayers
```

Each player has:

- `scope`, `firstAt`, `lastAt`, `activeFor` (`lastAt` minus `firstAt`)
- `tiles`: tiles it still holds — its take is the tile's latest and the paint is
  still there
- `takes`: every take it made, held or painted over since; a tile taken twice
  counts twice
- `tilesPerMinute` and `takesPerMinute`: each over `activeFor`, 0 for a single take
- `banned`/`bannedUntil`/`offence` when a ban is running

**High `takes` and `tiles` near zero is a bot being painted over as fast as it
paints.** The ledger keeps takes for 72h (`ledger.retention`) and survives a
restart. It keeps at most 4M takes (`ledgerStorage.maxTakes`); a busier stretch
drops the oldest first and logs `the ledger is full`.

Ban first, or the player repaints behind the revert. Leave out `duration` to
take the ladder's step (24h, 7 days, 3 years); it counts as an offence either
way. An address is banned as its scope:

```bash
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{"scope":"203.0.113.7"}' http://127.0.0.1:8081/planet.v1.AdminService/BanPlayer
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{"scope":"2001:db8:1:2::/64","duration":"3600s"}' http://127.0.0.1:8081/planet.v1.AdminService/BanPlayer
```

`"enforced":false` in the answer means `antiBot.shadowBan.enforce` is off: the
ban is kept but drops nothing. There is no unban call yet: see "Unban one scope"
in [Watching for bots](#6-watching-for-bots).

Then revert, dry run first:

```bash
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{"scope":"203.0.113.7","dryRun":true}' http://127.0.0.1:8081/planet.v1.AdminService/RevertPlayer
```

- `touched` is every tile the player took; `held` is those it still holds. Only
  `held` tiles change. A tile somebody took since stays theirs.
- Each goes back to what it held before the player's current run on it, or to
  nobody. When another player retook the tile in between, it goes back to that
  player's paint, not further: player il→ps, other ps→de, player de→ps gives
  `de`.
- Paced like the reassign, each tile an ordinary update on the live stream.
- A second run answers zeros: a reverted player has nothing left to revert.
- Every ban and revert is logged: `journalctl CONTAINER_NAME=cp-backend | grep "admin ban\|admin player revert"`.

### See how close the antibot is to one player

The `antibot ban` log line is only written when a ban fires. To see where a
player stands before that, inspect its scope (an address is read as its scope).
It changes nothing and is not logged:

```bash
docker compose exec backend wget -qO- --header 'Content-Type: application/json' --post-data '{"scope":"203.0.113.7"}' http://127.0.0.1:8081/planet.v1.AdminService/InspectPlayer
```

- `readings` has one entry per watchdog: `level` is `clear`, `suspect` or
  `certain`, and `evidence` is the rule and its numbers, as the ban line writes
  them. A `clear` watchdog has no evidence. A verdict older than
  `antiBot.jury.suspicionWindow` reads `clear`.
- `suspects` against `minSuspects`, and `guilty`: what the jury would decide if
  the player clicked now. One `certain` is enough alone.
- `clicks`, `activeFor`, `longestGap`, `lastClickAt`, `topCountry`: the same
  summary the ban line carries.
- `banned`, `bannedUntil`, `offence`, `flags` when a ban is running, enforced or not.
- `"tracked":false` means the jury has not seen the scope in
  `antiBot.jury.trackWindow`: it is not clicking now, or not from this scope.
- With `antiBot.enabled` off it is refused: `server returned error: HTTP/1.1 400`. A bad scope is refused the same way.

## Rollback

- **Bad backend build:** `BACKEND_IMAGE=ghcr.io/raphoester/clickplanet-backend:<sha>` in `.env`, then `docker compose up -d backend`.
- **Lost or corrupt tile state:** stop the backend, restore the `planet` schema from a dump (`drop schema planet cascade`, then `psql -U clickplanet clickplanet < planet-DATE.sql`), start it again.
- **Lost or corrupt chat messages:** the same, with the `chat` schema and `chat-DATE.sql`.
- **In-process storage misbehaving:** there is no config switch back to Redis — that code is gone. Roll the backend image back to a pre-migration `<sha>` and restore the matching Redis stack from git history.
- **Frontend:** roll back the deployment in the Pages dashboard.
