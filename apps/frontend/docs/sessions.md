# Sessions

The backend gates `Click` on a token it minted, and refuses one that carries
none. `session.ts` declares the contract — `SessionProvider`, the
`X-Session-Token` header name, `SessionUnavailableError`, and `NoSession` —
and `turnstileSession.ts` implements it in two halves that are worth keeping
apart:

- **`SessionClient`** holds one session and mints another when it is close to
  lapsing. No DOM, no script tag, no network of its own — which is why all of
  its behaviour is under test.
- **`turnstileAttester`** is the half that touches Cloudflare: it loads
  `challenges.cloudflare.com/turnstile/v0/api.js` on first use, renders a widget,
  resolves with its token and removes it again.

**It mints through `auth.v1.AuthService/CreateSession`**, which also gives the
browser an account: a guest one, kept in the `cp_sid` cookie the answer sets
(HttpOnly, on the API's host). The next mint sends the cookie back and gets the
same account. The page never reads the cookie; the menu learns what it holds
from `GetMe` — see [Sign-in](sign-in.md). `session.v1` is deprecated on the
backend and this build no longer calls it.

**`invalidate` also drops a mint in flight.** A sign-in or a sign-out changes
the account the cookie names, and a token minted before that would name the old
one for its hour. A generation counter keeps such a mint from being stored.

**`newAuthServiceClient` is the only transport that sends credentials.** Its
`fetch` wrapper adds `credentials: "include"`; without it connect-web sends
`same-origin`, and a cross-origin mint neither sends the cookie nor keeps the
one it is given — every mint would start a new guest. The click, map and chat
clients stay without it: who asks is in the token they carry, the identity
token included, and a read that carries a cookie is one no shared cache serves. Both halves are pinned in
`turnstileSession.test.ts`. A credentialed call needs the API to name the exact
origin and send `Access-Control-Allow-Credentials: true` — Caddy does in
production, and a local backend does from `httpServer.allowedOrigin`, which
must be the dev server's origin (`http://localhost:5173`).

**A held token outlives the page it was minted on.** `localTokenStore` keeps it
in local storage (`clickplanet-session`) and the client takes it back at
construction, by the same margin `held` applies to one it minted. Nothing mints
at load, so without this a reload held nothing until its first click and
everything that reads the token without minting read as a caller with no
account: the inventory came back empty, the meter showed the scope's bucket
rather than the player's, the stream followed the address, and presence listed
nobody. An invalidation and a failed mint both drop what was kept, so a reload
after a sign-out does not bring the old account's token back.

**Two tokens, one held at a time: the click token and the identity token.** The
click token comes from `CreateSession`, after a Turnstile check, and is the only
thing that may act. The identity token comes from `ResumeSession`, off the
`cp_sid` cookie with no Turnstile check: it names the same account and proves
no check, so the server takes it for reads alone (the backend's [The
click token and the identity token](../../backend/docs/sessions.md#the-click-token-and-the-identity-token)). A click token names the reader too, so
one token in hand serves both.

- **`token()`** answers the click token, minting through Turnstile when what is
  held is only an identity. **`held()`** never answers the identity token, so
  nothing that acts can send it by mistake: a click, a post, a reaction, a
  claim, an announce.
- **`identity()`** answers any fresh token, and otherwise resumes one silently;
  **`heldIdentity()`** is the same without the call. The reads use them: the
  history, the budget, the charges, both streams, the roster. `PlanetBackend`
  resumes at load (`followIdentity`), so a player back the next day is named
  from the first frame, with no Turnstile check and no click.
- **A cookie with no live session resumes nothing**: a first visit stays
  anonymous until its first click, as before. A resume that fails is no
  identity, logged, never an error a read would surface.
- **A resume that lands after a click token was minted leaves the click token**:
  it names the same account and proves more.
- **Kept like the click token**, with `identity: true` beside it, so a reload
  still knows a click must pass Turnstile. A kept token with no flag is a click
  token, as every token an older build kept was.

**It keeps the tokens and never the account.** The account is the `cp_sid`
cookie, which is HttpOnly and out of this page's reach either way. A token
lapses within the hour, is bound to the address that minted it, and a page that
could read this could mint one of its own off that cookie. A token restored on
another network is refused, which is the case a click already retries against a
fresh mint; a read that carries it answers for nobody, exactly as no token did.

**The store is injected, not read in the client.** `SessionClient` stays the
half with no DOM and no network — the same split as `turnstileAttester` — so
every rule about what is kept and when is under test without a browser.

**A fresh widget per attestation**, not one reset between uses. Turnstile tokens
are redeemed exactly once, and a widget that is created and destroyed has no
lifecycle left to get wrong.

**`appearance: "interaction-only"`** — the widget draws nothing for almost every
visitor. `.turnstile-host` in `index.css` is where it would appear if Turnstile
decides this one has to tick a box.

**Turnstile draws the checkbox inside a closed shadow root**, so no selector on
this page reaches it and no rule of ours styles it. That makes
`pointer-events: none` on the host unusable, however tempting: the property
*inherits* across the shadow boundary, and the `.turnstile-host iframe` rule
that would give it back matches nothing. A challenge styled that way is painted
on screen and passes every click straight through to the globe behind it — a
player who is asked to tick a box that cannot be ticked, and so cannot play.
Measured on the deployed site, not deduced.

Nothing is needed in its place: the host shrink-wraps the widget, and Turnstile
renders a **0x0** box while it is not challenging, so an idle host has no area
to swallow a click with. Its `z-index` is above every other layer — the chat
sheet is bottom-centre on a phone, exactly where the widget appears, and a
challenge is the one thing on the page that has to be answerable.

**Concurrent clicks share one mint.** A page load fires a flurry, and without
coalescing the first second of play would spend the whole per-IP mint budget on
one Turnstile round trip per click.

**Minting is refreshed a minute before the server would stop accepting the
token**, rather than after a refusal — a token that lapses between the check and
the server reading it costs a round trip the player can feel.

**A refused click is retried once against a freshly minted session, silently.**
A token that lapsed mid-session, or one bound to an address that changed when a
phone moved onto cellular, is not worth a dialog. The retry is not a loop: a
second `unauthenticated` is reported as `SessionUnavailableError` and raises
`SessionUnavailableModal`.

**Every failure inside the session client leaves as `SessionUnavailableError`.**
A refused mint answers `permission_denied`, which is also what a VPN-blocked
click answers — left bare it would reach the dialog telling the player to turn
off a VPN they may not be using.

**`VITE_TURNSTILE_SITEKEY` is what switches this on.** Unset, `main.tsx` wires
`NoSession` and the client sends no header, which is what a local backend with
`auth.enabled: false` expects. The sitekey is public — it is read off the
page — and useless without the secret, which only the backend holds. A server
that *enforces* sessions refuses every click from a build with no sitekey:
the two are configured together.

**It is a *build* variable, and Vite inlines it.** Setting it on the Workers
project changes nothing until a build runs: with it unset, `import.meta.env
.VITE_TURNSTILE_SITEKEY` folds to `undefined`, the `SessionClient` branch in
`main.tsx` becomes dead code, and Rollup drops `turnstileSession.ts` wholesale.
So a bundle built without it contains neither the sitekey nor the Turnstile
script URL — while still containing `X-Session-Token`, which comes from
`planetBackend.ts` and ships either way. That combination is the signature of a
stale build, not of a missing variable.

**The project only rebuilds on changes under `apps/frontend/`**, which is its
build watch path. A change to `deploy/` or `apps/backend/` that turns sessions
on server-side will therefore *not* ship a frontend that can mint one. Grep the
deployed bundle rather than trusting the dashboard:

```bash
B=$(curl -s https://clickplanet.lol/play | grep -oE '/assets/play-[A-Za-z0-9_-]+\.js' | head -1)
curl -s "https://clickplanet.lol$B" | grep -c 'challenges.cloudflare.com/turnstile'
```
