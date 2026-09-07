# Caddy with the Cloudflare DNS plugin.
#
# The stock caddy image cannot solve the ACME DNS-01 challenge — that needs a
# provider module compiled in. DNS-01 is what lets this box keep its origin IP
# private: Caddy proves domain control by writing a TXT record through the
# Cloudflare API instead of answering an inbound request on :80, so the A
# record can stay proxied (orange cloud) and port 80 never has to be reachable
# from the public internet.

FROM caddy:2.10-builder-alpine AS build

RUN xcaddy build --with github.com/caddy-dns/cloudflare

FROM caddy:2.10-alpine

COPY --from=build /usr/bin/caddy /usr/bin/caddy
