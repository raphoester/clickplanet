#!/usr/bin/env bash
#
# One-shot bootstrap for the production droplet.
#
# From your laptop, against a freshly created droplet:
#
#   ./bootstrap.sh --host 203.0.113.10 \
#     --api-domain api.clickplanet.lol \
#     --frontend-origin https://clickplanet.lol
#
# With --host the script copies itself to the box over SSH and re-runs itself
# there as root; everything below the "remote driver" section runs on the
# droplet. Without --host it assumes it is already on the box and provisions
# in place. The CI keypair is generated for you if it does not exist.
#
# Run it after the droplet exists and its A record is in place (see the DNS
# gate below for why that order matters).
#
# Idempotent: re-running upgrades the checkout and rolls the containers rather
# than duplicating anything. Safe to re-run after a partial failure.

set -euo pipefail

# ssh forwards the client's LC_* by default, and macOS sends LC_CTYPE="UTF-8",
# which is not a locale Linux has. Left alone, every apt and perl call on the
# box emits a dozen lines of locale warnings that bury the real output.
export LC_ALL=C.UTF-8
export LANG=C.UTF-8

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
CF_TOKEN=""
OPEN_ORIGIN=0
SWAP_SIZE="2G"
SSH_HOST=""
SSH_USER="root"
CI_KEY_PATH="${HOME}/.ssh/clickplanet_ci"
REMOTE_TMP="/tmp/clickplanet-bootstrap.sh"

# Arguments to hand to the copy of this script running on the droplet. --host,
# --ssh-user and the key flags are consumed here and re-derived there.
FORWARD=()

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m warn\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m fail\033[0m %s\n' "$*" >&2; exit 1; }

usage() {
	sed -n '3,20p' "$0" | sed 's/^# \{0,1\}//'
	cat <<'USAGE'

Options:
  --host ADDR                Provision this box over SSH from here. Omit when
                             running the script on the droplet itself.
  --ssh-user USER            User to connect as with --host (default: root).
  --api-domain DOMAIN        Hostname Caddy issues a certificate for. Required.
  --frontend-origin ORIGIN   Origin allowed to call the API, scheme included,
                             no trailing slash. Required.
  --ci-key "ssh-ed25519 ..." Public key GitHub Actions deploys with. Defaults
                             to ~/.ssh/clickplanet_ci.pub, generating the
                             keypair if it is absent.
  --no-ci-key                Skip the CI key; CI deploys will not work until
                             you install one by hand.
  --cf-token TOKEN           Cloudflare API token (Zone:DNS:Edit + Zone:Zone:
                             Read) Caddy uses for the ACME DNS-01 challenge.
                             Required. Read from $CLOUDFLARE_API_TOKEN if set.
  --open-origin              Allow 80/443 from anywhere instead of only
                             Cloudflare's ranges. Exposes the origin directly;
                             use only to debug past the proxy.
  --swap SIZE                Swap file to create if the box has none
                             (default 2G; "none" to skip).
  --backend-image REF        Pin a specific image instead of :latest.
  --skip-start               Provision only; do not bring the stack up.
  --skip-dns-check           Bypass the check that the record is proxied.
  --force-env                Overwrite an existing .env.
USAGE
}

while [[ $# -gt 0 ]]; do
	case "$1" in
		--host)            SSH_HOST="${2:-}"; shift 2 ;;
		--ssh-user)        SSH_USER="${2:-}"; shift 2 ;;
		--api-domain)      API_DOMAIN="${2:-}"; FORWARD+=(--api-domain "${2:-}"); shift 2 ;;
		--frontend-origin) FRONTEND_ORIGIN="${2:-}"; FORWARD+=(--frontend-origin "${2:-}"); shift 2 ;;
		--ci-key)          CI_KEY="${2:-}"; shift 2 ;;
		--ci-key-path)     CI_KEY_PATH="${2:-}"; shift 2 ;;
		--no-ci-key)       CI_KEY="none"; shift ;;
		--cf-token)        CF_TOKEN="${2:-}"; FORWARD+=(--cf-token "${2:-}"); shift 2 ;;
		--open-origin)     OPEN_ORIGIN=1; FORWARD+=(--open-origin); shift ;;
		--swap)            SWAP_SIZE="${2:-}"; FORWARD+=(--swap "${2:-}"); shift 2 ;;
		--backend-image)   BACKEND_IMAGE="${2:-}"; FORWARD+=(--backend-image "${2:-}"); shift 2 ;;
		--skip-start)      SKIP_START=1; FORWARD+=(--skip-start); shift ;;
		--skip-dns-check)  SKIP_DNS_CHECK=1; FORWARD+=(--skip-dns-check); shift ;;
		--force-env)       FORCE_ENV=1; FORWARD+=(--force-env); shift ;;
		-h|--help)         usage; exit 0 ;;
		*)                 die "unknown option: $1 (try --help)" ;;
	esac
done

# ------------------------------------------------------------ remote driver
#
# With --host, everything this script does happens on the droplet: it copies
# itself over and re-execs there. Nothing below this block runs on the laptop.

if [[ -n "$SSH_HOST" ]]; then
	[[ -n "$API_DOMAIN" ]] || die "--api-domain is required"
	[[ -n "$FRONTEND_ORIGIN" ]] || die "--frontend-origin is required"
	# Checked here as well as in the on-box preflight, so a typo fails before
	# the SSH round trip rather than after it.
	[[ "$FRONTEND_ORIGIN" =~ ^https?://[^/]+$ ]] \
		|| die "--frontend-origin must be scheme://host with no path or trailing slash, got: ${FRONTEND_ORIGIN}"
	if [[ -z "$CF_TOKEN" && -n "${CLOUDFLARE_API_TOKEN:-}" ]]; then
		log "using CLOUDFLARE_API_TOKEN from the environment"
		CF_TOKEN="$CLOUDFLARE_API_TOKEN"
		FORWARD+=(--cf-token "$CF_TOKEN")
	fi
	[[ -n "$CF_TOKEN" ]] || die "--cf-token is required.
       Create one at dash.cloudflare.com > My Profile > API Tokens > Custom:
         Permissions:    Zone / DNS / Edit   and   Zone / Zone / Read
         Zone Resources: Include / Specific zone / your zone
       Caddy uses it for the ACME DNS-01 challenge. That is what lets the A
       record stay proxied, so the origin IP never becomes public."
	command -v ssh >/dev/null || die "ssh not found on this machine"

	target="${SSH_USER}@${SSH_HOST}"

	# Resolve the key CI will deploy with. Generating it here rather than
	# asking for one is the difference between a one-liner and a checklist:
	# the private half never leaves the laptop, only the public half is sent.
	if [[ "$CI_KEY" == "none" ]]; then
		FORWARD+=(--no-ci-key)
	elif [[ -n "$CI_KEY" ]]; then
		FORWARD+=(--ci-key "$CI_KEY")
	else
		if [[ ! -f "${CI_KEY_PATH}.pub" ]]; then
			log "generating CI keypair at ${CI_KEY_PATH}"
			mkdir -p "$(dirname "$CI_KEY_PATH")"
			ssh-keygen -t ed25519 -f "$CI_KEY_PATH" -N "" -C "github-actions clickplanet" >/dev/null
		else
			log "using existing CI key ${CI_KEY_PATH}.pub"
		fi
		FORWARD+=(--ci-key "$(cat "${CI_KEY_PATH}.pub")")
	fi

	# accept-new records the host key on first contact, the way answering "yes"
	# would, but still refuses a *changed* key on a host we already know. Plain
	# StrictHostKeyChecking=no would accept both and give up MITM protection
	# for every later connection, including the ones CI makes.
	SSH_OPTS=(-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10)

	log "checking SSH to ${target}"
	# BatchMode first so an unreachable box fails fast instead of hanging on a
	# prompt; if that fails, try again interactively so a passphrase-protected
	# key or a password-auth box can still get through.
	if ! ssh "${SSH_OPTS[@]}" -o BatchMode=yes "$target" true 2>/dev/null; then
		ssh "${SSH_OPTS[@]}" "$target" true \
			|| die "$(printf 'cannot SSH to %s.\n       The droplet must be up and your own key attached to it (DigitalOcean\n       does that at creation time if you selected a key).' "$target")"
	fi

	log "copying bootstrap to ${SSH_HOST}"
	ssh "${SSH_OPTS[@]}" "$target" "cat > ${REMOTE_TMP} && chmod +x ${REMOTE_TMP}" < "$0"

	# printf %q so the public key, which contains spaces, survives the remote
	# shell intact.
	remote_args="$(printf '%q ' "${FORWARD[@]}")"

	log "running bootstrap on ${SSH_HOST}"
	rc=0
	ssh "${SSH_OPTS[@]}" "$target" "bash ${REMOTE_TMP} ${remote_args}" || rc=$?

	ssh "${SSH_OPTS[@]}" "$target" "rm -f ${REMOTE_TMP}" || true
	[[ $rc -eq 0 ]] || die "remote bootstrap failed (exit ${rc})"

	if [[ "$CI_KEY" != "none" ]]; then
		cat <<SECRETS

$(log "set the repository secrets so CI can deploy")

  gh secret set VPS_HOST    --body '${SSH_HOST}'
  gh secret set VPS_USER    --body 'deploy'
  gh secret set VPS_SSH_KEY < '${CI_KEY_PATH}'

SECRETS
	fi
	exit 0
fi

# ---------------------------------------------------------------- preflight

[[ $EUID -eq 0 ]] || die "run as root, or use --host to drive this from your laptop"
[[ -n "$API_DOMAIN" ]] || die "--api-domain is required"
[[ -n "$FRONTEND_ORIGIN" ]] || die "--frontend-origin is required"
[[ -n "$CI_KEY" ]] || die "--ci-key is required on the box (or pass --no-ci-key)"
[[ -n "$CF_TOKEN" ]] || die "--cf-token is required on the box"

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

# Everything below may drop to the deploy user via runuser, which inherits the
# current directory. Started over SSH that is /root, mode 700, which deploy
# cannot stat — and tools that stat "." (docker compose, git) then fail with
# errors that point nowhere near the real cause. Sit somewhere world-readable.
cd /

# ------------------------------------------------------------------ packages

export DEBIAN_FRONTEND=noninteractive

# ------------------------------------------------------------------- swap

# A $6 droplet ships with 1 GB of RAM and no swap at all, so a transient spike
# is an OOM kill rather than a slow moment. Nothing in this stack needs much
# memory at rest — the tile map is a few MB — but apt upgrades and docker
# unpacking layers both spike, and the OOM killer picks the biggest process,
# which is the API holding the game state.
if [[ "$SWAP_SIZE" == "none" ]]; then
	log "skipping swap (--swap none)"
elif [[ -n "$(swapon --show --noheadings 2>/dev/null)" ]]; then
	log "swap already active ($(free -h | awk '/Swap:/{print $2}'))"
else
	log "creating ${SWAP_SIZE} swap file"
	# fallocate is instant on ext4; dd is the portable fallback for filesystems
	# where a fallocated file cannot be used as swap.
	fallocate -l "$SWAP_SIZE" /swapfile 2>/dev/null \
		|| dd if=/dev/zero of=/swapfile bs=1M count="$(numfmt --from=iec "$SWAP_SIZE" | awk '{print int($1/1048576)}')" status=none
	chmod 600 /swapfile
	mkswap /swapfile >/dev/null
	swapon /swapfile
	grep -q '^/swapfile ' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
	# Swap here is an emergency buffer, not a place to page the working set to:
	# keep the kernel preferring RAM until it genuinely runs short.
	echo 'vm.swappiness=10' > /etc/sysctl.d/99-clickplanet-swap.conf
	sysctl -q -w vm.swappiness=10
fi

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

if [[ $OPEN_ORIGIN -eq 1 ]]; then
	warn "opening 80/443 to the world — the origin is reachable directly"
	ufw allow 80/tcp >/dev/null
	ufw allow 443/tcp >/dev/null
else
	# Only Cloudflare may reach the origin. Without this, hiding the IP behind
	# the proxy is decorative: anyone who learns the address (Censys and Shodan
	# index every cert served on :443) could still connect straight to the box.
	log "restricting 80/443 to Cloudflare's ranges"
	cf_ranges="$( { curl -fsS --max-time 15 https://www.cloudflare.com/ips-v4; echo;
	                curl -fsS --max-time 15 https://www.cloudflare.com/ips-v6; } 2>/dev/null | grep -E '[0-9a-fA-F:.]+/[0-9]+' || true)"
	[[ -n "$cf_ranges" ]] || die "could not fetch Cloudflare's IP ranges; re-run, or use --open-origin"

	# Drop any world-open rules a previous run left behind.
	ufw delete allow 80/tcp >/dev/null 2>&1 || true
	ufw delete allow 443/tcp >/dev/null 2>&1 || true

	while read -r cidr; do
		[[ -n "$cidr" ]] || continue
		ufw allow from "$cidr" to any port 80 proto tcp >/dev/null
		ufw allow from "$cidr" to any port 443 proto tcp >/dev/null
	done <<<"$cf_ranges"
	log "allowed $(grep -c . <<<"$cf_ranges") Cloudflare ranges"
fi

ufw --force enable >/dev/null

# ---------------------------------------------------------------- checkout

if [[ -d "${CHECKOUT}/.git" ]]; then
	log "updating existing checkout"
	# Pull as deploy, not root. The checkout is owned by deploy, and git's
	# safe.directory guard refuses to operate on a repository owned by another
	# user ("detected dubious ownership"). Pulling as root would also drop
	# root-owned objects into .git for the next run to trip over. This is the
	# same user CI pulls as, so both stay consistent.
	chown -R "$DEPLOY_USER:$DEPLOY_USER" "$CHECKOUT"
	runuser -u "$DEPLOY_USER" -- git -C "$CHECKOUT" pull --ff-only
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
	# A .env written before the DNS-01 switch has no token, and compose refuses
	# to start without it. Top it up rather than making the user hunt for this.
	if ! grep -q '^CLOUDFLARE_API_TOKEN=.' "$env_file"; then
		log "adding CLOUDFLARE_API_TOKEN to the existing .env"
		sed -i '/^CLOUDFLARE_API_TOKEN=/d' "$env_file"
		printf 'CLOUDFLARE_API_TOKEN=%s\n' "$CF_TOKEN" >> "$env_file"
	fi
else
	log "writing .env"
	cat > "$env_file" <<ENV
# Written by bootstrap.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)
API_DOMAIN=${API_DOMAIN}
FRONTEND_ORIGIN=${FRONTEND_ORIGIN}
CLOUDFLARE_API_TOKEN=${CF_TOKEN}
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

# With DNS-01 the certificate no longer depends on an inbound connection, so
# this is not about ACME any more — it is about reachability. ufw admits only
# Cloudflare, so the record MUST be proxied (orange cloud) or nothing reaches
# the origin at all. A record pointing straight at this box would also put the
# IP into public DNS, which is what this setup exists to avoid.
if [[ $SKIP_DNS_CHECK -eq 0 ]]; then
	log "checking how ${API_DOMAIN} resolves"
	public_ip="$(curl -fsS --max-time 5 http://169.254.169.254/metadata/v1/interfaces/public/0/ipv4/address 2>/dev/null \
		|| curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || true)"

	resolved="$(getent ahostsv4 "$API_DOMAIN" | awk '{print $1}' | sort -u || true)"
	if [[ -z "$resolved" ]]; then
		die "${API_DOMAIN} does not resolve yet.
       Add an A record for it in Cloudflare pointing at ${public_ip:-this box},
       leave the cloud icon ORANGE (proxied), then re-run."
	elif [[ -n "$public_ip" ]] && grep -qx "$public_ip" <<<"$resolved"; then
		if [[ $OPEN_ORIGIN -eq 1 ]]; then
			warn "${API_DOMAIN} points straight at this box; the origin IP is public"
		else
			die "${API_DOMAIN} resolves to ${public_ip}, this box's own address.
       That means the record is DNS only (grey cloud). Two problems: the origin
       IP is published in DNS, and ufw admits only Cloudflare, so visitors
       cannot reach it at all.

       Set that record to Proxied (orange cloud) in Cloudflare, confirm with
         dig +short ${API_DOMAIN}
       returning Cloudflare addresses (104.x / 172.6x), and re-run. The box is
       already provisioned; re-running redoes nothing."
		fi
	else
		log "DNS ok (proxied — origin IP not published)"
	fi
fi

# ------------------------------------------------------------------- image

# Both images are built in CI; nothing is compiled on this box.
for image in "${BACKEND_IMAGE:-ghcr.io/raphoester/clickplanet-backend:latest}" \
             "${CADDY_IMAGE:-ghcr.io/raphoester/clickplanet-caddy:latest}"; do
	log "checking ${image} is pullable"
	if ! docker pull -q "$image" >/dev/null 2>&1; then
		die "cannot pull ${image}.
       Run the \"deploy backend\" workflow once, then set that package to
       Public — GHCR creates packages private even for public repos."
	fi
done

# -------------------------------------------------------------------- start

# runuser resolves the group list at exec time, so this picks up the docker
# group added above without the re-login an interactive shell would need.
#
# cd first: this script runs as root from root's home over SSH, and runuser
# keeps the current directory. Compose stats "." while validating the file, so
# leaving the cwd at /root (mode 700) makes it fail as the deploy user with
# `error in parsing "compose-spec.json": stat .: permission denied` — which
# says nothing about permissions on the directory it is actually reading.
# Being in STACK_DIR also matches what CI does, so both derive the same
# project name ("vps") and therefore the same volume names.
cd "$STACK_DIR"

log "starting the stack"
runuser -u "$DEPLOY_USER" -- docker compose up -d

# ------------------------------------------------------------------- verify

log "waiting for the certificate (up to 3 min; DNS-01 waits on TXT propagation)"
for i in $(seq 1 60); do
	if curl -fsS --max-time 5 "https://${API_DOMAIN}/v2/rpc/map-density" >/dev/null 2>&1; then
		log "API is answering over HTTPS"
		break
	fi
	[[ $i -eq 60 ]] && warn "API not answering yet — check: docker compose -f ${STACK_DIR}/docker-compose.yaml logs caddy"
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
