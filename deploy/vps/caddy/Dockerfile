FROM caddy:2.11-builder-alpine AS build

RUN xcaddy build --with github.com/caddy-dns/cloudflare

FROM caddy:2.11-alpine

COPY --from=build /usr/bin/caddy /usr/bin/caddy
