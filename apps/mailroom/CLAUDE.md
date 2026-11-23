# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

The mailroom: a Cloudflare Worker in Go, built to WebAssembly, that takes the players' email replies to the operator's
answers and posts each one, raw, to the backend, then forwards to the operator's mailbox a mail the backend keeps in no
note. It decides nothing: all reading of a mail is in the backend. This file is only an index (see the root
[Docs rule](../../CLAUDE.md#docs-rule)).

- [The mailroom](docs/mailroom.md) — the `email` handler, what it posts and with which secret, what each answer of the
  backend makes it do (done, forward a stray mail, reject, try again), why a forward and not a stray mail kept in the
  backend, the workers-go commit it pins, its size against the free plan, how it is checked, built and deployed, and
  the Cloudflare set-up it needs.
