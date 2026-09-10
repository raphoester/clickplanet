# Production deploy: one DigitalOcean droplet + Cloudflare Pages

Replaces the old DigitalOcean stack (droplet + managed Redis + load balancer +
container registry, ~$50/month). The droplet was never the expensive part — the
managed add-ons were, and none of them are needed. Target cost is **~$6/month**.

| Piece | Where | Cost |
|---|---|---|
| Frontend (static bundle + textures) | Cloudflare Pages | free |
| API + WebSocket (`cmd/api`) | DigitalOcean droplet, Docker Compose | ~$6/mo |
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
- `docker-compose.yaml` — Caddy + backend. That is the whole stack; there is no database.
- `Caddyfile` — TLS via DNS-01, reverse proxy, CORS, WebSocket passthrough
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

WebSockets work through the proxy on the free plan.

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
Docker, creates the `deploy` user, generates and installs a CI keypair
(`~/.ssh/clickplanet_ci`, private half never leaves your laptop), restricts ufw
to SSH plus Cloudflare's ranges on 80/443, clones the repo to
`/opt/clickplanet`, writes `.env`, installs the nightly backup cron, builds
Caddy with the Cloudflare DNS plugin, and starts the stack. It finishes by
printing the `gh secret set` commands for step 5.

It is idempotent — re-running after a failure skips whatever is already done —
and it stops with a specific message rather than a confusing one when something
is not ready: the wrong CPU architecture, a token you did not pass, a
grey-clouded DNS record, or a backend image that is not pullable yet.

On first boot the API finds no snapshot and starts from an empty map, logging
`no tile snapshot found`. Check it:

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
game still runs — `session.enabled: false` keeps the old address-only
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
`session.turnstile.hostnames` are checked against the hostname siteverify
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
accept. Left empty the API generates one at boot and warns, which invalidates
every session in flight on each restart — every player then pays one extra round
trip on their next click. Rotating it deliberately costs the same and nothing
more.

`.env` is gitignored. `.env.example` beside it is the template and is committed.

### Roll it out in three steps

`backend.yaml` ships `session.enabled: false`, and `session.enforce: false`
under it. Nothing below breaks a running site at any point: the API starts
minting before anything requires a session, and starts requiring one only once
the clients that can mint are the overwhelming majority.

1. Set `session.enabled: true` in `backend.yaml` (leave `enforce` false) and
   deploy the backend. Then watch:

   ```bash
   docker compose exec backend wget -qO- localhost:8080/metrics | grep click_session_checks
   ```

   `verdict="missing"` is every click from a client that sends no token —
   at this point, all of them.

2. Deploy the frontend with `VITE_TURNSTILE_SITEKEY` set. `verdict="valid"`
   should climb and `missing` should fall away as caches expire.

3. Once `valid` is the overwhelming majority, set `session.enforce: true` in
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

`shadowBan` watches for one behaviour: taking a tile back moments after losing
it, over and over, in a band no hand holds. A flagged caller's clicks are
answered `OK` and dropped. That is deliberately not a refusal — a 403 names the
check that tripped, and a silent no-op names nothing, so working around it is
guesswork instead of a diff. It is not permanent: the caller reads the map back
over the same websocket and will notice eventually.

`backend.yaml` ships `enabled: true` with `detector.enforce: false`, which
measures and logs without dropping anything. **Do not flip `enforce` before
reading both of the following.**

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
never which caller owns it. Per-caller numbers come from the log below, and
only for callers that flag. That is why the bounds start wide.

### The log says who

The address is never a metric label — that is unbounded cardinality, and it
would put personal data in every scrape. It goes to the log instead:

```bash
docker compose logs backend | grep "shadowban candidate"
```

```
level=WARN msg="shadowban candidate" scope=198.51.100.20 reactions=20
  median=993.8ms spread=138.1ms topCountry=ps topCountryClicks=20 clicks=20
  tiles="[2013 2014 2015 2016 2017 2018 2019 2020]"
```

**`spread` is the number to judge on, not `median`.** A caller answering at
almost exactly one second, twenty times, within 138 ms of itself, is running a
timer — the delay is human-looking on purpose and only the regularity gives it
away. Compare that against the lines real players produce: they are named too at
these bounds, and their spread is far wider.

`topCountry` is the country the caller painted with most. It is **context, not
evidence** — the client declares it in the request, so it is changed by editing
one string, and plenty of real players paint the same flags a bot does. Read it
to understand what a caller was doing; never widen the rule to act on it.

Before enforcing, get into a tile war yourself and confirm your own line's
numbers sit clearly outside the ones you are about to set.

### Then turn it on

`backend.yaml` ships `maxMedian: 2s` / `maxSpread: 1s`, wide on purpose so the
log speaks. Tighten both to sit between the bot's line and the human ones, then
set `detector.enforce: true` and redeploy.

`shadowban_flagged` is how many callers are inside a ban and **counts while
`enforce` is false too** — a non-zero gauge in observe mode means the rule is
biting, not that anything was dropped. `shadowbanned_clicks` is the one that
stays at 0 until you enforce. Both are readable with the `wget` line above.

To undo one, set `enforce` back to false and redeploy — bans live in memory
only, so a restart clears every one of them.


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

`chat.enabled: true` in `backend.yaml` publishes two routes Caddy already
forwards: `/chat.v1.ChatService/` and `/ws/chat`. `SendMessage` is an
**unauthenticated public write endpoint** — anyone who can reach the API can
post, under any name — so the things that keep it usable are all config:

| Knob | Where | Default here |
|---|---|---|
| Kill switch | `chat.enabled` | on — flip it off and the routes 404 again |
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

**`chat.log` holds personal data.** One JSONL line per message with the sender's
IP beside their text, in the same `tile_state` volume as the snapshot — so the
nightly backup below now copies personal data too, and its own retention is
whatever you keep those tarballs for. `chat.storage.retention` (30 days) is a
policy decision, not a cache size; shorten it if you would rather hold less.

Turning chat off needs no rebuild and no image change: `chat.enabled: false` in
`backend.yaml` then `docker compose --env-file .env up -d backend`. Anything in
that file can also be overridden from the `environment:` block instead —
`cfgutil` reads env vars with `.` as the nesting delimiter, so the key is the
config path verbatim (`chat.enabled: "false"`), which is how `CHAT_TAG_SALT`
reaches `chat.service.tagSalt`.

## 9. Backups

The whole game state is one snapshot file in the `tile_state` volume, written
every 30s and on every clean shutdown. A nightly cron on the box is enough:

```bash
0 4 * * * docker run --rm -v vps_tile_state:/state -v /home/deploy/backups:/out alpine \
  tar czf /out/tiles-$(date +\%F).tar.gz -C /state .
```

DigitalOcean's droplet backups (+20% of the droplet price, so ~$1.20/mo) cover
the whole disk if you would rather not think about it.

## Rollback

- **Bad backend build:** `BACKEND_IMAGE=ghcr.io/raphoester/clickplanet-backend:<sha>` in `.env`, then `docker compose up -d backend`.
- **Lost or corrupt tile state:** stop the backend, drop the newest backup's `tiles.snapshot` into the `tile_state` volume, start it again. A snapshot the API cannot parse is not fatal — it logs and starts from an empty map, so a bad file degrades to a reset rather than a crash loop.
- **In-process storage misbehaving:** there is no config switch back to Redis — that code is gone. Roll the backend image back to a pre-migration `<sha>` and restore the matching Redis stack from git history.
- **Frontend:** roll back the deployment in the Pages dashboard.
