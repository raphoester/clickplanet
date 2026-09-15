#!/usr/bin/env bash
#
# Simulates the production rollout of the current branch over BASE_REF, with live traffic.
#
#   e2e/rollout/run.sh
#
# 1. build the backend image of BASE_REF (old) and of HEAD (new)
# 2. start the old production stack from BASE_REF's deploy/vps
# 3. seed its data files: tiles.snapshot, bans.jsonl, chat.log
# 4. start traffic: players clicking, a probe every 100ms
# 5. deploy like the CI job does: pull the checkout, then `docker compose up -d`
# 6. keep the traffic on, stop it, and check: downtime, errors after the deploy, lost clicks,
#    lost seed, postgres row count, the imported snapshot renamed, bans kept
# 7. restart the new backend and check the map comes back from postgres
#
# Differences from production, all in override.yaml: no Caddy, no Turnstile, shadow bans not
# enforced, json-file logs. The images are built here, so `docker compose pull` is replaced by
# pointing BACKEND_IMAGE at the new tag.
#
# Env: BASE_REF (origin/main), E2E_PORT (18080), TRAFFIC_BEFORE (20), TRAFFIC_AFTER (30),
#      PLAYERS (40), KEEP=1 to leave the stack and work dir up.

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO="$(git -C "$HERE" rev-parse --show-toplevel)"
BASE_REF="${BASE_REF:-origin/main}"
export E2E_PORT="${E2E_PORT:-18080}"
TRAFFIC_BEFORE="${TRAFFIC_BEFORE:-20}"
TRAFFIC_AFTER="${TRAFFIC_AFTER:-30}"
PLAYERS="${PLAYERS:-40}"
KEEP="${KEEP:-0}"

PROJECT="cprollout"
OLD_IMAGE="clickplanet-backend:e2e-old"
NEW_IMAGE="clickplanet-backend:e2e-new"
ADDR="http://127.0.0.1:${E2E_PORT}"
MAX_INDEX=257948

WORK="$(mktemp -d "${TMPDIR:-/tmp}/cprollout.XXXXXX")"
CHECKOUT="${WORK}/checkout"
STACK="${CHECKOUT}/deploy/vps"

log() { printf '\n\033[1;34m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31mfail\033[0m %s\n' "$*" >&2; exit 1; }
now_ms() { python3 -c 'import time; print(int(time.time()*1000))'; }

compose() {
	local files=(-f "${STACK}/docker-compose.yaml" -f "${HERE}/override.yaml")
	if grep -q '^  postgres:' "${STACK}/docker-compose.yaml"; then
		files+=(-f "${HERE}/override.postgres.yaml")
	fi
	docker compose -p "$PROJECT" --project-directory "$STACK" "${files[@]}" "$@"
}

state() { docker run --rm -v "${PROJECT}_tile_state:/state" alpine:3 "$@"; }

wait_ready() {
	local deadline=$((SECONDS + 90))
	until curl -sf -o /dev/null -X POST -H 'Content-Type: application/json' -H 'X-Real-IP: 198.51.100.250' \
		-d '{}' "${ADDR}/planet.v1.ClickService/MapDensity"; do
		((SECONDS < deadline)) || { compose logs --tail 50 backend; die "the backend did not answer within 90s"; }
		sleep 0.1
	done
}

set_env() {
	local key="$1" value="$2"
	grep -v "^${key}=" "${STACK}/.env" > "${STACK}/.env.next" || true
	printf '%s=%s\n' "$key" "$value" >> "${STACK}/.env.next"
	mv "${STACK}/.env.next" "${STACK}/.env"
}

cleanup() {
	local status=$?
	if [[ -n "${TRAFFIC_PID:-}" ]] && kill -0 "$TRAFFIC_PID" 2>/dev/null; then
		kill -KILL "$TRAFFIC_PID" 2>/dev/null || true
	fi
	if [[ "$KEEP" == "1" ]]; then
		log "KEEP=1: stack ${PROJECT} and ${WORK} left up"
	else
		compose down -v --remove-orphans >/dev/null 2>&1 || true
		git -C "$REPO" worktree remove --force "${WORK}/base" >/dev/null 2>&1 || true
		git -C "$REPO" worktree remove --force "${WORK}/head" >/dev/null 2>&1 || true
		rm -rf "$WORK"
	fi
	exit "$status"
}
trap cleanup EXIT

command -v docker >/dev/null || die "docker not found"
command -v go >/dev/null || die "go not found"

# ---------------------------------------------------------------- 1. images

log "building ${OLD_IMAGE} from ${BASE_REF} and ${NEW_IMAGE} from HEAD"
git -C "$REPO" worktree add --detach "${WORK}/base" "$BASE_REF" >/dev/null
git -C "$REPO" worktree add --detach "${WORK}/head" HEAD >/dev/null
docker build -q -t "$OLD_IMAGE" "${WORK}/base/apps/backend" >/dev/null
docker build -q -t "$NEW_IMAGE" "${WORK}/head/apps/backend" >/dev/null

(cd "$HERE" && go build -o "${WORK}/rollout" .)

# ------------------------------------------------------------ 2. old stack

log "starting the production stack of ${BASE_REF}"
mkdir -p "$CHECKOUT"
git -C "$REPO" archive "$BASE_REF" deploy/vps | tar -x -C "$CHECKOUT"

# What bootstrap.sh writes on the droplet. Caddy's values are never used: Caddy is not started.
cat > "${STACK}/.env" <<ENV
API_DOMAIN=e2e.invalid
FRONTEND_ORIGIN=http://e2e.invalid
CLOUDFLARE_API_TOKEN=unused
CHAT_TAG_SALT=$(openssl rand -hex 16)
SESSION_SECRET=$(openssl rand -hex 32)
TURNSTILE_SECRET=unused
BACKEND_IMAGE=${OLD_IMAGE}
ENV

compose up -d backend
wait_ready

# ------------------------------------------------------------------ 3. seed

log "seeding tiles.snapshot, bans.jsonl and chat.log"
mkdir -p "${WORK}/seed"
"${WORK}/rollout" seed -dir "${WORK}/seed" -max-index "$MAX_INDEX"

compose stop backend
docker run --rm -v "${PROJECT}_tile_state:/state" -v "${WORK}/seed:/seed:ro" alpine:3 \
	sh -c 'cp /seed/tiles.snapshot /seed/bans.jsonl /seed/chat.log /state/ && chown 1000:1000 /state/*'
compose start backend
wait_ready
"${WORK}/rollout" check-seed -addr "$ADDR" -seed "${WORK}/seed/seed.json" -max-index "$MAX_INDEX"

# --------------------------------------------------------------- 4. traffic

log "traffic on for ${TRAFFIC_BEFORE}s before the deploy"
MARKS="${WORK}/marks"
"${WORK}/rollout" traffic -addr "$ADDR" -seed "${WORK}/seed/seed.json" -marks "$MARKS" \
	-max-index "$MAX_INDEX" -players "$PLAYERS" &
TRAFFIC_PID=$!
sleep "$TRAFFIC_BEFORE"

# ---------------------------------------------------------------- 5. deploy

log "deploying HEAD: pull the checkout, then docker compose up -d"
git -C "$REPO" archive HEAD deploy/vps | tar -x -C "$CHECKOUT"

# The step the PR asks for before merging, then the image `docker compose pull` would fetch.
set_env POSTGRES_PASSWORD "$(openssl rand -hex 32)"
set_env BACKEND_IMAGE "$NEW_IMAGE"

echo "deploy_start $(now_ms)" > "$MARKS"
compose up -d backend
wait_ready
echo "deploy_end $(now_ms)" >> "$MARKS"

log "traffic on for ${TRAFFIC_AFTER}s after the deploy"
sleep "$TRAFFIC_AFTER"

# ----------------------------------------------------------------- 6. check

kill -INT "$TRAFFIC_PID"
deadline=$((SECONDS + 90))
while kill -0 "$TRAFFIC_PID" 2>/dev/null; do
	((SECONDS < deadline)) || die "the traffic check did not finish within 90s"
	sleep 1
done
if ! wait "$TRAFFIC_PID"; then
	TRAFFIC_PID=""
	die "the traffic check failed"
fi
TRAFFIC_PID=""

log "checking postgres, the imported snapshot and the bans file"
owned="$("${WORK}/rollout" dump -addr "$ADDR" -out "${WORK}/map.json" -max-index "$MAX_INDEX")"
sleep 3 # one flush interval, and then some
rows="$(compose exec -T postgres psql -U clickplanet -d clickplanet -tAc 'select count(*) from planet.tiles')"
[[ "$rows" == "$owned" ]] || die "postgres holds ${rows} tiles, the map ${owned}"
echo "postgres holds the ${rows} owned tiles of the map"

files="$(state ls /state)"
grep -qx 'tiles.snapshot.imported' <<<"$files" || die "tiles.snapshot was not renamed: ${files}"
! grep -qx 'tiles.snapshot' <<<"$files" || die "tiles.snapshot is still there"
echo "tiles.snapshot was renamed tiles.snapshot.imported"

bans="$(state cat /state/bans.jsonl)"
for scope in 192.0.2.10 192.0.2.11; do
	grep -q "\"scope\":\"${scope}\"" <<<"$bans" || die "the seeded ban on ${scope} is gone"
done
echo "the seeded bans are still in bans.jsonl"

# --------------------------------------------------------------- 7. restart

log "restarting the new backend: the map must come back from postgres"
compose restart backend
wait_ready
"${WORK}/rollout" compare -addr "$ADDR" -with "${WORK}/map.json" -max-index "$MAX_INDEX"

log "PASS"
