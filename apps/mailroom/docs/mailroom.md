# The mailroom

**A Cloudflare Worker, `clickplanet-mailroom`, written in Go and built to WebAssembly, with one `email` handler: a
dumb pipe.** The operator's answer to a note goes out with Reply-To `support+<note id>@clickplanet.lol` (see the
backend's [Support](../../backend/docs/support.md#answering-a-note)). Cloudflare Email Routing can only forward a mail,
drop it, or hand it to a Worker, so this Worker hands every mail to that address to the backend as it came. Reading the
mail (the note id, the words, the quoted history) is the backend's.

- **What it posts** (`deliver.go`, `Backend.Deliver`): `POST` to `MAILROOM_URL`
  (`https://api.clickplanet.lol/mailroom/replies`), the raw message as the body (`message/rfc822`), the envelope
  recipient in `X-Envelope-To` (`message.to`, which keeps its `+` part once Subaddressing is on), and
  `Authorization: Bearer <MAILROOM_SECRET>`.
- **What it answers Cloudflare**: a 2xx is done, kept or dropped by the backend. A 413 rejects the mail for good
  (`SetReject`): the sender is told it is too large. **Any other answer, or no answer, is an error**, and an error
  answers the sending server a temporary failure, so it tries again later: the backend down, restarting or refusing the
  secret loses no reply.
- **`main.go`** (built for `js/wasm` alone) registers the handler with
  [workers-go](https://github.com/syumai/workers-go)'s `exp/cloudflare/email`, reads `MAILROOM_URL` and
  `MAILROOM_SECRET` with `cloudflare.Getenv`, and posts through the Worker's `fetch`. **The email package is on
  workers-go's main branch, not in a release**: `go.mod` pins a commit (1 November 2026).
- **The secret** is `MAILROOM_SECRET`, a Worker secret, and the backend's `support.mailroom.secret`
  (`env://MAILROOM_SECRET`, the repository's Actions secret). Both must be the same.
- **`MAILROOM_URL`** is a plain var in `wrangler.jsonc`.
- **Checks**: `make vet` (the host and `js/wasm`), `make test` (`Deliver` against an `httptest` backend: what is
  posted, an error on a refusal or an unreachable backend, a rejection on 413), `make build`. CI runs all three on a
  change under `apps/mailroom/` (`.github/workflows/check-mailroom.yml`).
- **The build** (`make build`): workers-go's `workers-assets-gen` writes the JavaScript glue under `build/`, and Go
  builds `build/app.wasm`, stripped (`-trimpath -ldflags="-s -w"`): about 11 MB, **2.8 MB gzipped, close to the free
  Workers plan's 3 MB cap**. A dependency that grows it past the cap fails the deploy.
- **Deploy**: `make deploy` (`wrangler deploy`, which runs `make build` itself), by hand: it changes rarely. Its own
  build, from this directory alone (see [Independence of the apps](../../../docs/app-independence.md)).

## Cloudflare set-up

- **Email Routing on `clickplanet.lol`** (its MX already point to Cloudflare), with **Subaddressing** on, so
  `support+<id>@` reaches the rule of `support@`.
- **A routing rule `support@clickplanet.lol` → the Worker `clickplanet-mailroom`.**
- **Email Sending sends from `support@clickplanet.lol`**: the domain is onboarded, and the REST API takes any sender on
  it.
