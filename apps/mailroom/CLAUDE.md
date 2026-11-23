# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

The mailroom: a Cloudflare Worker in Go, built to WebAssembly, that takes the players' email replies to the operator's answers and posts each one,
raw, to the backend. It decides nothing: all reading of a mail is in Go. This file is only an index (see the root
[Docs rule](../../CLAUDE.md#docs-rule)).

- [The mailroom](docs/mailroom.md) — the `email` handler, what it posts and with which secret, what makes Cloudflare
  try again, the workers-go commit it pins, its size against the free plan, how it is checked, built and deployed,
  and the Cloudflare set-up it needs.
