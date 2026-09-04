# Production deploy: one small VPS + Cloudflare Pages

Replaces the DigitalOcean droplet + managed Redis + load balancer + container
registry setup. Target cost is **~€4/month** (the VPS) — everything else is on
free tiers.

| Piece | Where | Cost |
|---|---|---|
| Frontend (static bundle + textures) | Cloudflare Pages | free |
| API + WebSocket (`cmd/api`) | Hetzner CX22, Docker Compose | ~€3.79/mo |
| TLS | Caddy, automatic Let's Encrypt | free |
| Images | GitHub Container Registry | free |
| DNS | Cloudflare | free |

Prices are from late 2025 — check before you commit.

## Layout

- `docker-compose.yaml` — Caddy + backend (+ `redis` profile for rollback)
- `Caddyfile` — TLS, reverse proxy, CORS, WebSocket passthrough
- `backend.yaml` — API config; secrets come from env, not this file
- `.env.example` — copy to `.env` on the box

`deploy/docker-compose.yaml` (one level up) stays as the *local* full-stack
compose. This directory is only for the production box.

## 1. DNS

Two records on `clickplanet.lol`:

| Name | Type | Value | Proxy |
|---|---|---|---|
| `api` | A | VPS IPv4 | **DNS only (grey cloud)** |
| `@` / `www` | CNAME | Pages target | Proxied (orange) |

Keep `api` unproxied at first — Caddy needs a direct connection on :80 to issue
its certificate. You can switch it to proxied afterwards if you want Cloudflare
in front of the API too; WebSockets work on the free plan.

## 2. The box

Hetzner CX22 (2 vCPU / 4 GB, x86). **Pick x86, not ARM** — `apps/backend/Dockerfile`
pins `GOARCH=amd64`.

```bash
ssh root@YOUR_IP
adduser --disabled-password --gecos "" deploy && usermod -aG docker deploy
curl -fsSL https://get.docker.com | sh
ufw allow OpenSSH && ufw allow 80 && ufw allow 443 && ufw enable
git clone https://github.com/raphoester/clickplanet.git /opt/clickplanet
chown -R deploy:deploy /opt/clickplanet
```

Then as `deploy`:

```bash
cd /opt/clickplanet/deploy/vps
cp .env.example .env && $EDITOR .env   # set API_DOMAIN, FRONTEND_ORIGIN, REDIS_PASSWORD
docker compose --profile redis up -d
```

The `redis` profile is needed **until the in-process storage PR lands**. After
that: switch `backend.yaml` to the memory driver, then `docker compose up -d`
without the profile and `docker rm -f cp-redis`.

One-time Lua script load (Redis path only), then paste the sha1 into
`backend.yaml` and restart the backend:

```bash
docker exec -i cp-redis sh -c 'redis-cli -a "$REDIS_PASSWORD" -x script load < /static/setAndPublishOnStream.lua'
```

Check it:

```bash
curl -sS https://api.clickplanet.lol/v2/rpc/map-density | head -c 200
```

## 3. Frontend on Cloudflare Pages

Create a Pages project from the GitHub repo:

| Setting | Value |
|---|---|
| Root directory | `apps/frontend` |
| Build command | `npm ci && npm run build` |
| Output directory | `dist` |
| Env var | `VITE_API_BASE_URL=https://api.clickplanet.lol` |

Generated protobuf code is committed, so the build needs no `buf`.

`npm run build` now also copies `static/` into `dist/static/`, skipping the
24 MB `coordinates.json` — that file is bundled into the JS at build time and
never fetched at runtime, so uploading it would waste a quarter of the deploy.

**Watch the bundle size.** `coordinates.json` is imported into the main chunk,
which currently builds to **23.5 MiB against Cloudflare's 25 MiB per-file
limit**. There is ~1.5 MiB of headroom. If a dependency or a denser tile map
pushes it over, Pages will reject the upload — the fix is to load the
coordinates as a binary `Float32Array` fetched at runtime (~5 MB instead of
25 MB, and no JSON parse), which means making the viewer's point setup async.

## 4. CI

`.github/workflows/deploy-backend.yml` builds the image to GHCR and rolls the
container over SSH. Repository secrets required:

- `VPS_HOST` — the box's IP
- `VPS_USER` — `deploy`
- `VPS_SSH_KEY` — private key whose public half is in `deploy`'s `authorized_keys`

Pages deploys itself on push; no workflow needed.

## 5. Backups

The whole game state is one snapshot file in the `tile_state` volume (or
`redis_data` on the Redis path). A nightly cron on the box is enough:

```bash
0 4 * * * docker run --rm -v vps_tile_state:/state -v /home/deploy/backups:/out alpine \
  tar czf /out/tiles-$(date +\%F).tar.gz -C /state .
```

Hetzner's automated backups (+20% of the instance price) cover the whole disk
if you would rather not think about it.

## Rollback

- **Bad backend build:** `BACKEND_IMAGE=ghcr.io/raphoester/clickplanet-backend:<sha>` in `.env`, then `docker compose up -d backend`.
- **In-process storage misbehaving:** restore the `redis:` block in `backend.yaml` and `docker compose --profile redis up -d`.
- **Frontend:** roll back the deployment in the Pages dashboard.
