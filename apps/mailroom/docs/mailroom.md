# The mailroom

**A Cloudflare Worker, `clickplanet-mailroom`, with one `email` handler: a dumb pipe.** The operator's answer to a note
goes out with Reply-To `support+<note id>@clickplanet.lol` (see the backend's
[Support](../../backend/docs/support.md#answering-a-note)). Cloudflare Email Routing hands every mail to that address
to this Worker, and the Worker posts it to the backend as it came. Reading the mail (the note id, the words, the quoted
history) is the backend's, in Go.

- **What it posts** (`src/index.ts`, `deliver`): `POST` to `MAILROOM_URL`
  (`https://api.clickplanet.lol/mailroom/replies`), the raw message as the body (`message/rfc822`), the envelope
  recipient in `X-Envelope-To` (`message.to`, which keeps its `+` part once Subaddressing is on), and
  `Authorization: Bearer <MAILROOM_SECRET>`.
- **What it answers Cloudflare**: a 2xx is done, kept or dropped by the backend. A 413 rejects the mail for good
  (`setReject`): the sender is told it is too large. **Anything else throws**, and a throw answers the sending server a
  temporary failure, so it tries again later: the backend down, restarting or refusing the secret loses no reply.
- **The secret** is `MAILROOM_SECRET`, a Worker secret (`wrangler secret put MAILROOM_SECRET`), and the backend's
  `support.mailroom.secret` (`env://MAILROOM_SECRET`, the repository's Actions secret). Both must be the same.
- **`MAILROOM_URL`** is a plain var in `wrangler.jsonc`.
- **Checks**: `npm run typecheck`, `npm test` (vitest, with `fetch` stubbed: what is posted, a throw on a refusal or an
  unreachable backend, a rejection on 413). CI runs both on a change under `apps/mailroom/`
  (`.github/workflows/check-mailroom.yml`).
- **Deploy**: `npm run deploy` (`wrangler deploy`), by hand: it changes rarely. Its own build, from this directory
  alone (see [Independence of the apps](../../../docs/app-independence.md)).

## Cloudflare set-up

- **Email Routing on `clickplanet.lol`** (its MX already point to Cloudflare), with **Subaddressing** on, so
  `support+<id>@` reaches the rule of `support@`.
- **A routing rule `support@clickplanet.lol` → the Worker `clickplanet-mailroom`.**
- **Email Sending allows `support@clickplanet.lol`** as a sender, for the answers.
