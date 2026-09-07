#!/usr/bin/env bash
#
# One-shot bootstrap for the production droplet.
#
#   sudo bash bootstrap.sh \
#     --api-domain api.clickplanet.lol \
#     --frontend-origin https://clickplanet.lol \
#     --ci-key "$(cat ~/.ssh/clickplanet_ci.pub)"
#
# Run as root on a fresh Ubuntu LTS box, after the droplet exists and its A
# record is in place (see the DNS gate below for why that order matters).
#
# Idempotent: re-running upgrades the checkout and rolls the containers rather
# than duplicating anything. Safe to re-run after a partial failure.

set -euo pipefail

REPO_URL="https://github.com/raphoester/clickplanet.git"
CHECKOUT="/opt/clickplanet"
STACK_DIR="${CHECKOUT}/deploy/vps"
DEPLOY_USER="deploy"
BACKUP_DIR="/home/${DEPLOY_USER}/backups"

API_DOMAIN=""
FRONTEND_ORIGIN=""
CI_KEY=""
BACKEND_IMAGE=""
SKIP_START=0
SKIP_DNS_CHECK=0
FORCE_ENV=0

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m warn\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m fail\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
	sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'
	cat <<'USAGE'

Options:
  --api-domain DOMAIN        Hostname Caddy issues a certificate for. Required.
  --frontend-origin ORIGIN   Origin allowed to call the API, scheme included,
                             no trailing slash. Required.
  --ci-key "ssh-ed25519 ..." Public key GitHub Actions deploys with. Required
                             unless --no-ci-key.
  --no-ci-key                Skip the CI key; CI deploys will not work until
                             you install one by hand.
  --backend-image REF        Pin a specific image instead of :latest.
  --skip-start               Provision only; do not bring the stack up.
  --skip-dns-check           Bypass the DNS gate. Risks burning Let's Encrypt
                             failed-validation rate limits.
  --force-env                Overwrite an existing .env.
USAGE
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--api-domain)      API_DOMAIN="${2:-}"; shift 2 ;;
		--frontend-origin) FRONTEND_ORIGIN="${2:-}"; shift 2 ;;
		--ci-key)          CI_KEY="${2:-}"; shift 2 ;;
		--no-ci-key)       CI_KEY="none"; shift ;;
		--backend-image)   BACKEND_IMAGE="${2:-}"; shift 2 ;;
		--skip-start)      SKIP_START=1; shift ;;
		--skip-dns-check)  SKIP_DNS_CHECK=1; shift ;;
		--force-env)       FORCE_ENV=1; shift ;;
		-h|--help)         usage; exit 0 ;;
		*)                 die "unknown option: $1 (try --help)" ;;
	esac
done

# ---------------------------------------------------------------- preflight

[[ $EUID -eq 0 ]] || die "run as root (sudo bash $0 ...)"
[[ -n "$API_DOMAIN" ]] || die "--api-domain is required"
[[ -n "$FRONTEND_ORIGIN" ]] || die "--frontend-origin is required"
[[ -n "$CI_KEY" ]] || die "--ci-key is required (or pass --no-ci-key)"

# FRONTEND_ORIGIN is compared byte-for-byte against the browser's Origin header,
# so a trailing slash or a missing scheme silently breaks CORS in the browser
# while curl without an Origin header keeps working.
[[ "$FRONTEND_ORIGIN" =~ ^https?://[^/]+$ ]] \
	|| die "--frontend-origin must be scheme://host with no path or trailing slash, got: ${FRONTEND_ORIGIN}"

# apps/backend/Dockerfile pins GOARCH=amd64 and CI publishes only that arch, so
# the image cannot run here at all on an ARM droplet. Fail now rather than at
# the first docker pull.
arch="$(uname -m)"
[[ "$arch" == "x86_64" ]] \
	|| die "this box is ${arch}; the backend image is amd64-only, rebuild the droplet as Intel/AMD"

log "preflight ok (${arch}, $(. /etc/os-release && echo "$PRETTY_NAME"))"

# ------------------------------------------------------------------ packages

export DEBIAN_FRONTEND=noninteractive

if ! command -v docker >/dev/null 2>&1; then
	log "installing docker"
	curl -fsSL https://get.docker.com | sh
else
	log "docker already present ($(docker --version))"
fi

if ! command -v git >/dev/null 2>&1; then
	log "installing git"
	apt-get update -qq && apt-get install -y -qq git
fi

# --------------------------------------------------------------- deploy user

if ! id -u "$DEPLOY_USER" >/dev/null 2>&1; then
	log "creating ${DEPLOY_USER} user"
	adduser --disabled-password --gecos "" "$DEPLOY_USER"
else
	log "${DEPLOY_USER} user already exists"
fi
usermod -aG docker "$DEPLOY_USER"

# The account is --disabled-password, so this key is the only way in. Without
# it GitHub Actions cannot connect and the deploy job fails on auth.
if [[ "$CI_KEY" != "none" ]]; then
	ssh_dir="/home/${DEPLOY_USER}/.ssh"
	auth="${ssh_dir}/authorized_keys"
	install -d -m 700 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$ssh_dir"
	touch "$auth"
	if grep -qxF "$CI_KEY" "$auth"; then
		log "CI key already authorised"
	else
		log "installing CI key"
		printf '%s\n' "$CI_KEY" >> "$auth"
	fi
	chown "$DEPLOY_USER:$DEPLOY_USER" "$auth"
	chmod 600 "$auth"
else
	warn "no CI key installed — GitHub Actions deploys will fail until you add one"
fi

# ---------------------------------------------------------------- firewall

# OpenSSH is allowed before enable, or this locks us out of our own box.
log "configuring firewall"
ufw allow OpenSSH >/dev/null
ufw allow 80/tcp >/dev/null
ufw allow 443/tcp >/dev/null
ufw --force enable >/dev/null
ufw status | sed 's/^/     /'

# ---------------------------------------------------------------- checkout

if [[ -d "${CHECKOUT}/.git" ]]; then
	log "updating existing checkout"
	git -C "$CHECKOUT" pull --ff-only
else
	log "cloning ${REPO_URL}"
	git clone --depth 1 "$REPO_URL" "$CHECKOUT"
fi
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$CHECKOUT"

[[ -f "${STACK_DIR}/docker-compose.yaml" ]] \
	|| die "${STACK_DIR}/docker-compose.yaml missing — wrong branch or bad clone?"

# --------------------------------------------------------------------- .env

env_file="${STACK_DIR}/.env"
if [[ -f "$env_file" && $FORCE_ENV -eq 0 ]]; then
	log ".env already exists, leaving it alone (--force-env to overwrite)"
else
	log "writing .env"
	cat > "$env_file" <<ENV
# Written by bootstrap.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)
API_DOMAIN=${API_DOMAIN}
FRONTEND_ORIGIN=${FRONTEND_ORIGIN}
BACKEND_IMAGE=${BACKEND_IMAGE:-ghcr.io/raphoester/clickplanet-backend:latest}
ENV
	chown "$DEPLOY_USER:$DEPLOY_USER" "$env_file"
	chmod 600 "$env_file"
fi

# ----------------------------------------------------------------- backups

# The snapshot in the tile_state volume is the entire game state and the only
# thing on this box worth backing up.
if ! crontab -u "$DEPLOY_USER" -l 2>/dev/null | grep -q 'vps_tile_state'; then
	log "installing nightly tile-state backup cron"
	install -d -m 755 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$BACKUP_DIR"
	{
		crontab -u "$DEPLOY_USER" -l 2>/dev/null || true
		echo "0 4 * * * docker run --rm -v vps_tile_state:/state -v ${BACKUP_DIR}:/out alpine tar czf /out/tiles-\$(date +\\%F).tar.gz -C /state ."
		echo "30 4 * * * find ${BACKUP_DIR} -name 'tiles-*.tar.gz' -mtime +14 -delete"
	} | crontab -u "$DEPLOY_USER" -
else
	log "backup cron already installed"
fi

if [[ $SKIP_START -eq 1 ]]; then
	log "provisioning done (--skip-start), stack not started"
	exit 0
fi

# --------------------------------------------------------------- DNS gate

# Caddy attempts ACME issuance the instant it starts. Let's Encrypt rate-limits
# failed validations per hostname per hour, so starting before the A record
# resolves can lock the domain out of certificate issuance for an hour on a box
# that is otherwise fine. Waiting costs nothing; not waiting can cost an hour.
if [[ $SKIP_DNS_CHECK -eq 0 ]]; then
	log "checking ${API_DOMAIN} resolves to this box"
	public_ip="$(curl -fsS --max-time 5 http://169.254.169.254/metadata/v1/interfaces/public/0/ipv4/address 2>/dev/null \
		|| curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"
	[[ -n "$public_ip" ]] || die "could not determine this box's public IP (pass --skip-dns-check to bypass)"

	resolved="$(getent ahostsv4 "$API_DOMAIN" | awk '{print $1}' | sort -u || true)"
	if [[ -z "$resolved" ]]; then
		die "${API_DOMAIN} does not resolve yet. Add the A record -> ${public_ip} (grey cloud), wait, re-run."
	elif ! grep -qx "$public_ip" <<<"$resolved"; then
		die "$(printf '%s resolves to %s but this box is %s.\n       If that is a Cloudflare IP the record is orange-clouded — Caddy needs\n       a direct connection on :80 to pass the ACME challenge. Set it to DNS only.' \
			"$API_DOMAIN" "$(tr '\n' ' ' <<<"$resolved")" "$public_ip")"
	fi
	log "DNS ok (${API_DOMAIN} -> ${public_ip})"
fi

# ------------------------------------------------------------------- image

image="${BACKEND_IMAGE:-ghcr.io/raphoester/clickplanet-backend:latest}"
log "checking ${image} is pullable"
if ! docker pull -q "$image" >/dev/null 2>&1; then
	die "$(printf 'cannot pull %s.\n       Run the "deploy backend" workflow once, then make the GHCR package\n       public — GHCR creates packages private even for public repos.' "$image")"
fi

# -------------------------------------------------------------------- start

# runuser resolves the group list at exec time, so this picks up the docker
# group added above without the re-login an interactive shell would need.
log "starting the stack"
runuser -u "$DEPLOY_USER" -- docker compose --project-directory "$STACK_DIR" up -d

# ------------------------------------------------------------------- verify

log "waiting for the certificate (up to 90s)"
for i in $(seq 1 30); do
	if curl -fsS --max-time 5 "https://${API_DOMAIN}/v2/rpc/map-density" >/dev/null 2>&1; then
		log "API is answering over HTTPS"
		break
	fi
	[[ $i -eq 30 ]] && warn "API not answering yet — check: docker compose -f ${STACK_DIR}/docker-compose.yaml logs caddy"
	sleep 3
done

# The header this checks is the one failure that only ever shows up in a
# browser: curl without an Origin header passes happily either way.
cors="$(curl -fsSI --max-time 5 -H "Origin: ${FRONTEND_ORIGIN}" \
	"https://${API_DOMAIN}/v2/rpc/map-density" 2>/dev/null \
	| tr -d '\r' | awk -F': ' 'tolower($1)=="access-control-allow-origin"{print $2}')"
if [[ "$cors" == "$FRONTEND_ORIGIN" ]]; then
	log "CORS ok (Access-Control-Allow-Origin: ${cors})"
else
	warn "expected Access-Control-Allow-Origin: ${FRONTEND_ORIGIN}, got: '${cors:-<none>}'"
fi

cat <<DONE

$(log "bootstrap complete")

  stack     ${STACK_DIR}
  logs      sudo -u ${DEPLOY_USER} docker compose --project-directory ${STACK_DIR} logs -f
  state     docker volume inspect vps_tile_state
  backups   ${BACKUP_DIR} (nightly 04:00 UTC, pruned after 14 days)

Next: point Cloudflare Pages at apps/frontend with
VITE_API_BASE_URL=https://${API_DOMAIN} (no trailing slash).
DONE
