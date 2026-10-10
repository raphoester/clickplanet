FROM golang:1.27-alpine3.24 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

WORKDIR /root
COPY . .

ENV GOCACHE=/root/.cache/go-build
ENV GOPATH=/root/go
RUN --mount=type=cache,target="/root/.cache/go-build" \
	--mount=type=cache,target="/root/go/pkg/mod" \
    go build -o /root/bin/ops ./cmd/ops;


# Debian, not alpine: journalctl ships with systemd, and an older one cannot read a newer host's journal.
FROM debian:trixie-slim AS cmd

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates systemd \
	&& rm -rf /var/lib/apt/lists/*

ENV OPS_LISTEN_ADDRESS=:8080
ENV OPS_JOURNAL_DIRECTORY=/var/log/journal
ENV OPS_ACCESS_LOG_DIRECTORY=/var/log/caddy

COPY --from=build /root/bin/ops /usr/local/bin/ops

CMD ["ops"]
