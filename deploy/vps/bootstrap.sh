#!/usr/bin/env bash

set -euo pipefail

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

REPO_URL="https://github.com/raphoester/clickplanet.git"
CHECKOUT="/opt/clickplanet"
STACK_DIR="${CHECKOUT}/deploy/vps"
DEPLOY_USER="deploy"

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

FORWARD=()

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m warn\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m fail\033[0m %s\n' "$*" >&2; exit 1; }

random_salt() {
	openssl rand -hex 16 2>/dev/null \
		|| od -An -tx1 -N16 /dev/urandom | tr -d ' \n'
}

random_secret() {
	openssl rand -hex 32 2>/dev/null \
		|| od -An -tx1 -N32 /dev/urandom | tr -d ' \n'
}

usage() {
	cat <<'USAGE'
Usage: bootstrap.sh [--host ADDR] --api-domain DOMAIN --frontend-origin ORIGIN [options]

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
  --force-env                Replace the secrets already on the box with fresh
                             ones, instead of keeping what is there.
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

if [[ -n "$SSH_HOST" ]]; then
	[[ -n "$API_DOMAIN" ]] || die "--api-domain is required"
	[[ -n "$FRONTEND_ORIGIN" ]] || die "--frontend-origin is required"
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

	# accept-new, never no: a changed host key must still be refused.
	SSH_OPTS=(-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10)

	log "checking SSH to ${target}"
	if ! ssh "${SSH_OPTS[@]}" -o BatchMode=yes "$target" true 2>/dev/null; then
		ssh "${SSH_OPTS[@]}" "$target" true \
			|| die "$(printf 'cannot SSH to %s.\n       The droplet must be up and your own key attached to it (DigitalOcean\n       does that at creation time if you selected a key).' "$target")"
	fi

	log "copying bootstrap to ${SSH_HOST}"
	ssh "${SSH_OPTS[@]}" "$target" "cat > ${REMOTE_TMP} && chmod +x ${REMOTE_TMP}" < "$0"

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

	cat <<PUSH
$(log "push the box's secrets up, BEFORE the first deploy")

  ssh ${DEPLOY_USER}@${SSH_HOST} 'grep -hE "^(CLOUDFLARE_API_TOKEN|CHAT_TAG_SALT|SESSION_SECRET|TURNSTILE_SECRET|POSTGRES_PASSWORD|GOOGLE_CLIENT_SECRET|DISCORD_CLIENT_SECRET)=." ${STACK_DIR}/.env ${STACK_DIR}/.env.caddy ${STACK_DIR}/.env.backend' \\
    | sort -u | while IFS='=' read -r name value; do \\
        printf '%s' "\$value" | gh secret set "\$name" --repo raphoester/clickplanet && echo "set \$name"; \\
      done

  Actions secrets cannot be read back, so keep a copy in a password manager too.

PUSH
	exit 0
fi

[[ $EUID -eq 0 ]] || die "run as root, or use --host to drive this from your laptop"
[[ -n "$API_DOMAIN" ]] || die "--api-domain is required"
[[ -n "$FRONTEND_ORIGIN" ]] || die "--frontend-origin is required"
[[ -n "$CI_KEY" ]] || die "--ci-key is required on the box (or pass --no-ci-key)"
[[ -n "$CF_TOKEN" ]] || die "--cf-token is required on the box"

[[ "$FRONTEND_ORIGIN" =~ ^https?://[^/]+$ ]] \
	|| die "--frontend-origin must be scheme://host with no path or trailing slash, got: ${FRONTEND_ORIGIN}"

arch="$(uname -m)"
[[ "$arch" == "x86_64" ]] \
	|| die "this box is ${arch}; the backend image is amd64-only, rebuild the droplet as Intel/AMD"

log "preflight ok (${arch}, $(. /etc/os-release && echo "$PRETTY_NAME"))"

# runuser keeps the cwd, and deploy cannot stat /root (mode 700).
cd /

export DEBIAN_FRONTEND=noninteractive

tmp_err="$(mktemp)"
trap 'rm -f "$tmp_err"' EXIT

if [[ "$SWAP_SIZE" == "none" ]]; then
	log "skipping swap (--swap none)"
elif [[ -n "$(swapon --show --noheadings 2>/dev/null)" ]]; then
	log "swap already active ($(free -h | awk '/Swap:/{print $2}'))"
else
	log "creating ${SWAP_SIZE} swap file"
	fallocate -l "$SWAP_SIZE" /swapfile 2>/dev/null \
		|| dd if=/dev/zero of=/swapfile bs=1M count="$(numfmt --from=iec "$SWAP_SIZE" | awk '{print int($1/1048576)}')" status=none
	chmod 600 /swapfile
	mkswap /swapfile >/dev/null
	swapon /swapfile
	grep -q '^/swapfile ' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
	echo 'vm.swappiness=10' > /etc/sysctl.d/99-clickplanet-swap.conf
	sysctl -q -w vm.swappiness=10
fi

if ! command -v docker >/dev/null 2>&1; then
	log "installing docker"
	curl -fsSL https://get.docker.com | sh
else
	log "docker already present ($(docker --version))"
fi

# docker-proxy hides the Cloudflare peer from Caddy, so Cf-Connecting-Ip would be ignored.
daemon_json=/etc/docker/daemon.json
if [[ ! -s "$daemon_json" ]]; then
	log "disabling docker's userland proxy (restarts docker)"
	mkdir -p /etc/docker
	echo '{ "userland-proxy": false }' > "$daemon_json"
	systemctl restart docker
elif grep -Eq '"userland-proxy"[[:space:]]*:[[:space:]]*false' "$daemon_json"; then
	log "docker userland proxy already disabled"
else
	die "${daemon_json} exists without \"userland-proxy\": false. Add it, then run: systemctl restart docker"
fi

if ! command -v git >/dev/null 2>&1; then
	log "installing git"
	apt-get update -qq && apt-get install -y -qq git
fi

if ! command -v jq >/dev/null 2>&1; then
	log "installing jq"
	apt-get update -qq && apt-get install -y -qq jq
fi

if ! id -u "$DEPLOY_USER" >/dev/null 2>&1; then
	log "creating ${DEPLOY_USER} user"
	adduser --disabled-password --gecos "" "$DEPLOY_USER"
else
	log "${DEPLOY_USER} user already exists"
fi
usermod -aG docker "$DEPLOY_USER"

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

log "configuring firewall"
ufw allow OpenSSH >/dev/null

if [[ $OPEN_ORIGIN -eq 1 ]]; then
	warn "opening 80/443 to the world — the origin is reachable directly"
	ufw allow 80/tcp >/dev/null
	ufw allow 443/tcp >/dev/null
else
	log "restricting 80/443 to Cloudflare's ranges"
	cf_ranges="$( { curl -fsS --max-time 15 https://www.cloudflare.com/ips-v4; echo;
	                curl -fsS --max-time 15 https://www.cloudflare.com/ips-v6; } 2>/dev/null | grep -E '[0-9a-fA-F:.]+/[0-9]+' || true)"
	[[ -n "$cf_ranges" ]] || die "could not fetch Cloudflare's IP ranges; re-run, or use --open-origin"

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

if [[ -d "${CHECKOUT}/.git" ]]; then
	log "updating existing checkout"
	chown -R "$DEPLOY_USER:$DEPLOY_USER" "$CHECKOUT"
	runuser -u "$DEPLOY_USER" -- git -C "$CHECKOUT" pull --ff-only
else
	log "cloning ${REPO_URL}"
	git clone --depth 1 "$REPO_URL" "$CHECKOUT"
fi
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$CHECKOUT"

[[ -f "${STACK_DIR}/docker-compose.yaml" ]] \
	|| die "${STACK_DIR}/docker-compose.yaml missing — wrong branch or bad clone?"

env_public="${STACK_DIR}/env.public"
[[ -f "$env_public" ]] || die "${env_public} missing — wrong branch or bad clone?"

public_value() { sed -n "s/^${1}=//p" "$env_public" | head -1; }
for pair in "API_DOMAIN:${API_DOMAIN}" "FRONTEND_ORIGIN:${FRONTEND_ORIGIN}"; do
	name="${pair%%:*}"; passed="${pair#*:}"; in_file="$(public_value "$name")"
	[[ -n "$in_file" ]] || die "${name} is missing from env.public"
	[[ "$in_file" == "$passed" ]] || die "${name} is ${in_file} in env.public but ${passed} was passed in.
  Both reach the same stack, so they cannot differ. Edit env.public (it is in
  git, and the deploy reads it) or pass the value it already holds."
done

existing_secret() {
	local name="$1" f
	for f in "${STACK_DIR}/.env.backend" "${STACK_DIR}/.env.caddy" "${STACK_DIR}/.env"; do
		[[ -f "$f" ]] || continue
		local value
		value="$(sed -n "s/^${name}=//p" "$f" | head -1)"
		[[ -n "$value" ]] && { printf '%s' "$value"; return; }
	done
}

secret_value() {
	local name="$1" generator="${2:-}" current=""
	if [[ $FORCE_ENV -eq 0 ]]; then
		current="$(existing_secret "$name")"
	fi
	if [[ -n "$current" ]]; then
		printf '%s' "$current"
	elif [[ -n "$generator" ]]; then
		log "generating ${name}" >&2
		"$generator"
	else
		log "leaving ${name} empty — set it in the Actions secrets, or on the box until the first deploy" >&2
	fi
}

log "rendering .env, .env.caddy and .env.backend"
SECRETS_JSON="$(jq -n \
	--arg CLOUDFLARE_API_TOKEN "$CF_TOKEN" \
	--arg CHAT_TAG_SALT "$(secret_value CHAT_TAG_SALT random_salt)" \
	--arg SESSION_SECRET "$(secret_value SESSION_SECRET random_secret)" \
	--arg POSTGRES_PASSWORD "$(secret_value POSTGRES_PASSWORD random_secret)" \
	--arg TURNSTILE_SECRET "$(secret_value TURNSTILE_SECRET)" \
	--arg GOOGLE_CLIENT_SECRET "$(secret_value GOOGLE_CLIENT_SECRET)" \
	--arg DISCORD_CLIENT_SECRET "$(secret_value DISCORD_CLIENT_SECRET)" \
	--arg CLOUDFLARE_EMAIL_TOKEN "$(secret_value CLOUDFLARE_EMAIL_TOKEN)" \
	'$ARGS.named')" \
	"${STACK_DIR}/render-env.sh" "$STACK_DIR"

if [[ -n "$BACKEND_IMAGE" ]]; then
	log "pinning BACKEND_IMAGE to ${BACKEND_IMAGE} until the first deploy"
	printf 'BACKEND_IMAGE=%s\n' "$BACKEND_IMAGE" >> "${STACK_DIR}/.env"
fi

for f in .env .env.caddy .env.backend; do
	chown "$DEPLOY_USER:$DEPLOY_USER" "${STACK_DIR}/${f}"
	chmod 600 "${STACK_DIR}/${f}"
done

if [[ $SKIP_START -eq 1 ]]; then
	log "provisioning done (--skip-start), stack not started"
	exit 0
fi

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

log "verifying the Cloudflare token can solve DNS-01"
command -v python3 >/dev/null || die "python3 is missing on this box; it is needed to read the Cloudflare API response"

cf_api() { curl -sS -H "Authorization: Bearer ${CF_TOKEN}" -H "Content-Type: application/json" "$@"; }

zones_json="$(cf_api "https://api.cloudflare.com/client/v4/zones?per_page=50" 2>&1 || true)"

zone_id="$(printf '%s' "$zones_json" | python3 -c '
import json, sys
raw = sys.stdin.read()
try:
    d = json.loads(raw)
except Exception:
    sys.stderr.write("Cloudflare returned something that is not JSON:\n  " + raw[:300] + "\n")
    sys.exit(1)
if not d.get("success"):
    errs = d.get("errors") or [{"message": "unknown error"}]
    sys.stderr.write("Cloudflare rejected the request:\n")
    for e in errs:
        sys.stderr.write("  [%s] %s\n" % (e.get("code", "?"), e.get("message", "")))
    sys.exit(2)
host = sys.argv[1]
zones = d.get("result", [])
best, zid = "", ""
for z in zones:
    n = z.get("name", "")
    if (host == n or host.endswith("." + n)) and len(n) > len(best):
        best, zid = n, z.get("id", "")
if not zid:
    names = ", ".join(z.get("name", "?") for z in zones) or "(none)"
    sys.stderr.write("The token works, but none of the zones it can see cover %s.\n" % host)
    sys.stderr.write("  zones visible to this token: %s\n" % names)
    sys.exit(3)
print(zid)
' "$API_DOMAIN" 2>"$tmp_err")" || {
	sed 's/^/       /' "$tmp_err" >&2
	die "the Cloudflare token cannot be used for DNS-01 (see above).
       It needs Zone / Zone / Read and Zone / DNS / Edit, with Zone Resources
       including the zone that owns ${API_DOMAIN}."
}

probe="_acme-challenge-bootstrap-check.${API_DOMAIN}"
create_json="$(cf_api -X POST "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records" \
	--data "{\"type\":\"TXT\",\"name\":\"${probe}\",\"content\":\"clickplanet bootstrap check\",\"ttl\":60}" 2>&1 || true)"

record_id="$(printf '%s' "$create_json" | python3 -c '
import json, sys
raw = sys.stdin.read()
try:
    d = json.loads(raw)
except Exception:
    sys.stderr.write("Cloudflare returned something that is not JSON:\n  " + raw[:300] + "\n")
    sys.exit(1)
if not d.get("success"):
    for e in (d.get("errors") or [{"message": "unknown error"}]):
        sys.stderr.write("  [%s] %s\n" % (e.get("code", "?"), e.get("message", "")))
    sys.exit(2)
print(d.get("result", {}).get("id", ""))
' 2>"$tmp_err")" || {
	sed 's/^/       /' "$tmp_err" >&2
	die "the token can read the zone but cannot create a DNS record there.
       Add the Zone / DNS / Edit permission — Caddy writes a TXT record every
       time the certificate renews."
}

cf_api -X DELETE "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records/${record_id}" >/dev/null 2>&1 || \
	warn "could not delete the probe record ${probe}; remove it by hand"

log "Cloudflare token ok (zone found, TXT write succeeded)"

for image in "${BACKEND_IMAGE:-ghcr.io/raphoester/clickplanet-backend:latest}" \
             "${CADDY_IMAGE:-ghcr.io/raphoester/clickplanet-caddy:latest}"; do
	log "checking ${image} is pullable"
	if ! docker pull -q "$image" >/dev/null 2>&1; then
		die "cannot pull ${image}.
       Run the \"deploy backend\" workflow once, then set that package to
       Public — GHCR creates packages private even for public repos."
	fi
done

cd "$STACK_DIR"

log "starting the stack"
runuser -u "$DEPLOY_USER" -- docker compose up -d

log "reloading the Caddyfile"
for i in $(seq 1 15); do
	runuser -u "$DEPLOY_USER" -- docker compose exec -T caddy \
		caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile >/dev/null 2>&1 && break
	[[ $i -eq 15 ]] && die "caddy reload failed — check: docker compose exec caddy caddy validate --config /etc/caddy/Caddyfile"
	sleep 2
done

log "waiting for the certificate (up to 3 min; DNS-01 waits on TXT propagation)"
for i in $(seq 1 60); do
	if curl -fsS --max-time 5 "https://${API_DOMAIN}/planet.v1.ClickService/MapDensity?connect=v1&encoding=json&message=%7B%7D" >/dev/null 2>&1; then
		log "API is answering over HTTPS"
		break
	fi
	[[ $i -eq 60 ]] && warn "API not answering yet — check: docker compose -f ${STACK_DIR}/docker-compose.yaml logs caddy"
	sleep 3
done

cors="$(curl -fsS -o /dev/null -D- --max-time 5 -H "Origin: ${FRONTEND_ORIGIN}" \
	"https://${API_DOMAIN}/planet.v1.ClickService/MapDensity?connect=v1&encoding=json&message=%7B%7D" 2>/dev/null \
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

Next: point Cloudflare Pages at apps/frontend with
VITE_API_BASE_URL=https://${API_DOMAIN} (no trailing slash).
DONE
