# The mailroom

**A Cloudflare Worker, `clickplanet-mailroom`, written in Go and built to WebAssembly, with one `email` handler: a
dumb pipe.** The operator's answer to a note goes out with Reply-To `support+<note id>@clickplanet.lol` (see the
backend's [Support](../../backend/docs/support.md#answering-a-note)). Cloudflare Email Routing can only forward a mail,
drop it, or hand it to a Worker, so this Worker hands every mail to that address, and to `support@` itself, to the
backend as it came. Reading the mail (the note id, the words, the quoted history) is the backend's. The Worker only
does what the backend's answer says.

- **What it posts** (`deliver.go`, `Backend.Deliver`): `POST` to `MAILROOM_URL`
  (`https://api.clickplanet.lol/mailroom/replies`), the raw message as the body (`message/rfc822`), the envelope
  recipient in `X-Envelope-To` (`message.to`, which keeps its `+` part once Subaddressing is on), and
  `Authorization: Bearer <MAILROOM_SECRET>`.
- **What it does with the answer**:
  - **202: kept** in its note (or kept already). Done.
  - **Any other 2xx: a stray mail**, which no note keeps: a player who wrote to `support@` itself, a reply to a note
    with no address any more, a mail the backend cannot read. The Worker **forwards it to `OPERATOR_ADDRESS`**
    (`Forward`), the operator's own mailbox, as it came. The backend answers 200. Only a 202 ends a mail without the
    operator seeing it, so a wrong `MAILROOM_URL` that answers 200 loses nothing either.
  - **413: rejected for good** (`SetReject`): the sender is told it is too large.
  - **Any other answer, or no answer, is an error**, and an error answers the sending server a temporary failure, so
    it tries again later: the backend down, restarting or refusing the secret loses no reply. A `Forward` that fails is
    an error too.
- **`main.go`** (built for `js/wasm` alone) registers the handler with
  [workers-go](https://github.com/syumai/workers-go)'s `exp/cloudflare/email`, reads `MAILROOM_URL`, `MAILROOM_SECRET`
  and `OPERATOR_ADDRESS` with `cloudflare.Getenv`, and posts through the Worker's `fetch`. **The email package is on
  workers-go's main branch, not in a release**: `go.mod` pins a commit (1 November 2026).
- **The secret** is `MAILROOM_SECRET`, a Worker secret, and the backend's `support.mailroom.secret`
  (`env://MAILROOM_SECRET`, the repository's Actions secret). Both must be the same.
- **`OPERATOR_ADDRESS`** is a Worker secret too (`wrangler secret put OPERATOR_ADDRESS`): not a secret, but a mailbox,
  and this repository is public. It must be a **verified destination address** of the account's Email Routing, or
  `Forward` fails.
- **`MAILROOM_URL`** is a plain var in `wrangler.jsonc`. `workers_dev` and `preview_urls` are off: the Worker has no page.
- **Checks**: `make vet` (the host and `js/wasm`), `make test` (`Deliver` against an `httptest` backend: what is
  posted, a 202 done with, another 2xx forwarded, an error on a refusal, an unreachable backend or a failed forward, a
  rejection on 413), `make build`. CI runs all three on a change under `apps/mailroom/`
  (`.github/workflows/check-mailroom.yml`).
- **The build** (`make build`): workers-go's `workers-assets-gen` writes the JavaScript glue under `build/`, and Go
  builds `build/app.wasm`, stripped (`-trimpath -ldflags="-s -w"`): about 11 MB, **2.8 MB gzipped, close to the free
  Workers plan's 3 MB cap**. A dependency that grows it past the cap fails the deploy.
- **Its cost, measured on 2026-11-23** (`wrangler tail` on two real mails): 11 ms of CPU warm, 56 ms on a cold start,
  against 1 ms for a JavaScript pipe. Both ran to the end, with no CPU-limit kill.
- **Deploy**: `make deploy` (`wrangler deploy`, which runs `make build` itself), by hand: it changes rarely. Its own
  build, from this directory alone (see [Independence of the apps](../../../docs/app-independence.md)).

## Why a forward, and not a stray mail kept in the backend

A stray mail is rare, and most of it is spam. Kept in the backend, it would be a concept of its own (a mail with no
note: its table, its store, its prune of the sender's address at 90 days, a read and a backoffice view), and the spam
with it, unfiltered. Forwarded, it is one status and one branch, and the operator's mailbox filters the spam. The cost:
the operator answers it from that mailbox, not from the backoffice.

## Cloudflare set-up

- **Email Routing on `clickplanet.lol`** (its MX already point to Cloudflare), with **Subaddressing** on, so
  `support+<id>@` reaches the rule of `support@`.
- **A routing rule `support@clickplanet.lol` → the Worker `clickplanet-mailroom`.**
- **`OPERATOR_ADDRESS` is a verified destination address** (Email Routing, Destination addresses).
- **Email Sending sends from `support@clickplanet.lol`**: the domain is onboarded, and the REST API takes any sender on
  it.
