# Sign-in

Optional from end to end: a player who never signs in plays exactly as before,
on the guest account the mint gives every browser. Signing in with Google,
Discord or a code sent to an email address keeps that account on every device.
On the CrazyGames page the only way in is CrazyGames' own, and the page follows
whoever CrazyGames says is signed in: see [CrazyGames](crazygames.md).

- `backends/account.ts` — the contract: `AccountBackend`, `Provider` (and
  `OAuthProvider`, the ones the page leaves for: every one but `email`), and
  `AuthError`, whose `failure` says why the server said no. **One class and not
  one per reason**, unlike a refused click: every one is shown the same way, as
  one line beside the button that was pressed.
- `backends/accountBackend.ts` — `ConnectAccountBackend`, over the client
  `newAuthServiceClient` builds, so every call carries the cookie. It maps
  Connect codes: `unimplemented` → `off`, `invalid_argument` → `notOffered`,
  `resource_exhausted` → `tooManyTries`, `failed_precondition` → `startAgain`,
  `permission_denied` → `refused`, `unauthenticated` → `notSignedIn`, anything
  else → `failed`. A refused link is matched on its `LinkRefusal` detail, not
  its code: `linkedElsewhere` or `alreadyLinked`. **Only the two reads are retried**: a retried
  `CompleteSignIn` would spend a code that is good once.
- `domain/signInCallback.ts` — `callbackOf`, what the provider sent to
  `/auth/callback`: a code and a state, a refusal (`error`, which wins), or a
  link with a part missing.
- `backends/player.ts` / `playerBackend.ts` — the username. `ConnectPlayerBackend`
  sends the click token with every call, and a call refused `unauthenticated` is
  sent once more with a fresh session, as a click is. `invalid_argument` →
  `invalid`, `already_exists` → `taken`, `permission_denied` → `guest`, a second
  `unauthenticated` → `notSignedIn`, a `RenameRefusal` detail →
  `RenameTooSoonError` (`tooSoon`, with `renamableAt`), anything else (a mint
  that failed included) → `failed`. Only `GetProfile` is retried.
- `app/account/accountStore.ts` — `AccountStore`, the section's state machine:
  `loading`, `hidden`, or `ready` with the offered providers, the linked ones,
  the action in flight and the last failure, and the username with its own save
  in flight and its own failure. No DOM and no network of its own, like
  `SessionClient`, so every transition is under test.
- `app/account/` — the rest is React: `AccountPanel` (the menu's You tab, named
  "Sign in" for a guest; a sheet on a phone), `DeleteAccountModal`,
  `SignInCallback` and `SignInGate`.

**The buttons come from `GetSignInOptions`**, which answers the providers the
server offers and is not throttled. Production runs with `auth.signIn.enabled`
off, so the list is empty and the menu looks as it did. **Never probe with
`StartSignIn` instead**: it spends the mint budget, which `CreateSession` needs.
A server without the RPC, or with the whole auth module off, answers 404, which
reads as no provider. So does any other failure of that read: a sign-in that
cannot say what it offers is better absent. A linked account still shows when
nothing is offered, so a player can sign out after sign-in is turned off.

**Sign-in shares the mint budget** (one every 30s, ten in hand): `StartSignIn`
and `CompleteSignIn` each spend one. `tooManyTries` says to wait a minute.

**The flow:**

1. "Sign in with Google" calls `StartSignIn` with the intent `signIn`, keeps
   the provider and the intent in session storage (`rememberedSignIn.ts`) and
   sends the browser to the URL it answers. "Link Discord" (`AccountStore.link`)
   sends the intent `link`. **The intent matters**: a sign-in with an identity
   another account uses moves the browser to that account, and a link is
   refused instead, so the player stays on the account they linked from.
2. The provider sends the browser to `/auth/callback?code=…&state=…`. The
   Workers asset handler serves `auth/callback.html` there, a copy of the game
   the build writes (see [Pages and routes](routes.md); `nginx.conf` has
   a route of its own), and the project's build watch path, `apps/frontend/`,
   covers every file involved.
3. **The code must not leak.** An inline script at the top of `play.html` adds
   `<meta name="referrer" content="no-referrer">` on that path before any other
   request is made, `public/_headers` sends the same `Referrer-Policy`, and
   `main.tsx` takes the query out of the address bar with `history.replaceState`
   before it renders anything.
4. `SignInGate` renders `SignInCallback` instead of the game. It calls
   `CompleteSignIn` **once** — a ref guards it, since StrictMode runs the effect
   twice and a second trade would fail and hide the first one's success.
5. On success, `AccountStore.completeSignIn` **invalidates the click token** and
   reads the account again, and the gate swaps in the game in place, at `/play`,
   with no reload. The next click mints a token that carries the new account.
   `CompleteSignIn` answers the outcome (`SignInOutcome`); a `linked` one that
   leaves the account one provider gave a guest its first identity, and the
   store marks it `guestLinked` (see [The refill a first sign-in earns](#the-refill-a-first-sign-in-earns)).
6. On failure the page says why in one line. `retryOf` picks what "Try again"
   does: send the same code again when the server did not use it
   (`tooManyTries`, `failed` — a spent budget is refused before the code is
   read), or go back to the remembered provider when the code is spent
   (`startAgain`, `refused`). With no remembered provider there is only "Back to
   the game".
7. A refused link (`linkedElsewhere`, `alreadyLinked`) is titled "Not linked"
   and offers only "Back to the game": the same identity would be refused
   again. The server changed nothing, so the player is still on their account.
   `linkedElsewhere` tells them how to move the identity: sign in with it,
   delete that account, then link it here.

**A signed-in player is given a username when it signs in** (the server draws
one, see the backend's [Player](../../backend/docs/player.md)), **and may pick another** in `AccountPanel`: 3 to 15
code points of letters of any script, digits, `_` and single spaces, not
starting with `guest_` in any case, unique ignoring case (the server's rule is
`players.NameOf`, see the backend's [Player](../../backend/docs/player.md)). `usernameOf` puts the draft in
NFC and cuts the spaces at its ends, as the server does, and that is what is
sent. `isValidUsername` mirrors the rule for the Save button with `\p{…}`
classes, and counts code points (`[...name].length`), never `length`. The input's
`maxLength` counts UTF-16 units, so it is twice the rule: the rule is the bound.
One part is the server's alone: JavaScript cannot name a character's script, so
the client refuses only Latin, Greek and Cyrillic mixed (the lookalikes), and a
name mixing other scripts comes back `invalid`. The server is the
authority and alone knows what is taken. The chat shows it (see [Live
chat](chat.md)). **A name changes at most once every 30 days**, a name given
at sign-in aside (the backend's [Player](../../backend/docs/player.md)). `GetProfile` and `SetName`
answer when it may change again (`Profile.renamableAt`), and so does a refusal
(`RenameTooSoonError`); `AccountStore` keeps it, and until then the form is
disabled and its hint says the date and time instead of the rule. The client
never counts the 30 days itself. **It is read after the account, not with it**: `GetProfile`
needs a click token, which can mean a mint, so the section shows as soon as
`GetMe` answers and the name follows. Only a linked account reads it — a guest
has none and should not mint to learn that — and a failed read leaves the name
unknown with the form still there. A save and the other actions never run at
once. A sign-in reads it again; a sign-out or a delete forgets it, and a read or
a save that lands after the account changed is dropped.

**A player with a username picks its name color** in `AccountPanel`, under the
username: the 12 of `NAME_COLORS` in a `role="group"` named "Name color", each
`aria-pressed`, none pressed before the first pick. There is no way back to no
color. `AccountStore.setColor` shows the pick at once, sends `SetColor` and
keeps the color the server answers, or the old one on a refusal; `GetProfile` answers it with the name, and `readProfile` reads
both. A color is refused without a username (`FailedPrecondition` → `unnamed`),
which is why the picker only shows with one. `usePresence` announces again once
the color held still for a second (`SETTLE_MS`), so the roster line follows.
Under the color, the same player has the "Show my name in clips" switch, which the clips service keeps, not
the account (see [In a clip](clip-appearances.md)).

**A signed-in account's panel has two tabs**: Progress, open first (see
[Titles](titles.md)), and Account, which holds the username, the color, linking
and signing out. A guest has no tabs: its panel is the sign-in buttons alone,
and it reads no titles.

**Signing in by email stays on the page.** The server offers `email` beside the
providers when `auth.email.enabled` is on, and `EmailSignIn` draws it under the
provider buttons, in `AccountPanel` and in `SignInPitchModal`: an address, then
a six-digit code. There is no callback page and nothing is remembered across a
trip, so the steps are store state: `AccountStore.sendCode` asks for a code and
keeps `code: {address, intent}` in the ready state, `checkCode` sends what was
typed (digits only), and `cancelCode` goes back to the address.

- **Each `StartEmailSignIn` needs a fresh Turnstile token**, since each sends an
  email: `ConnectAccountBackend` takes the same attester as `SessionClient`,
  and calls it once per code. Without a sitekey it sends an empty token, which
  a local backend with Turnstile off accepts.
- **A wrong code keeps the code step** (`wrongCode`), and so does a spent mint
  budget or a failure on the way. A lapsed code, or five wrong ones, is
  `newCode`: the step closes and the player asks again. Refused addresses come
  back with their reason in an `EmailRefusal` detail: `invalidEmail` or
  `disposableEmail`. `tooManyCodes` is the address's budget or the network's.
- **On success the account is read again** and the click token invalidated,
  exactly as after `CompleteSignIn`. An email account is linked: it clicks
  faster and is given a username it may change.
- To try it locally, run the backend with `auth.email.enabled` and
  `auth.email.delivery: log` (no Cloudflare account needed): the code is in the
  server log.

**A guest out of clicks is asked to sign in** (`SignInNudgeModal`, opened by
`useSignInNudge` in `Viewer`). Newcomers leave at the tap that finds the bank
empty, whatever its size (the backend's [A guest's first sign-in](../../backend/docs/sign-in-refill.md)),
so that tap is the moment. "Out of clicks", then three lines: a full bank now
(only while the server's `BonusRules.signInRefill` says so), clicks refill
`linkedMultiplier` times faster, and the land on every device; then the sign-in
buttons (`SignInButtons`, which `SignInPitchModal` uses too).

- **It opens on a dry tap**: a tap the globe refuses for an empty bank
  (`refusals` from `useGlobe`), from a guest the server offers sign-in to, with
  its multiplier, on a map that is not frozen, holding no refill (a refill is a
  full bank already, and the server gives a guest one at most).
- **It never covers a quiz.** A young account's first dry click brings its
  welcome quiz within about a second (the backend's [A newcomer's first quiz](../../backend/docs/welcome-bonus.md)).
  So the nudge waits `NUDGE_AFTER_MS` (1.5s) after the dry tap, and gives up if a
  quiz is up then or comes in the wait. The next dry tap after the quiz tries
  again: the quiz closing is no dry tap, as the bank may have refilled by then.
- **Once a page.** Closed, it stays closed until the next load. Signed in by
  email inside it, it closes as the player stops being a guest.

### The refill a first sign-in earns

**The server grants a guest's first sign-in one refill, and the game spends it
at once** (`useSignInRefill` in `Viewer`), so a guest who signed in at an empty
bank comes back to a full one. Planet grants it a second or two after the
sign-in, when player has made the account a player, so the hook reads the charges at once, after 2s and after
5s (`refreshCharges`, the read a paid mission uses), and spends the refill the
first time one is held (`useRefill`, the slot's own call; nothing on a full
bank). After `SIGN_IN_REFILL_WAIT_MS` (10s) it stops: a refill held later is the
player's to spend. It does nothing unless the store marked `guestLinked` and the
rules say a first sign-in earns one. The refill goes through `UseRefill` with a
fresh click token, which is linked, so it fills the account's own bank.

**Every way out of an account invalidates the click token too**: sign out, sign
out everywhere, and delete. The player plays on, and the next click mints a new
guest. `unauthenticated` on one of these means the account was already gone, and
is treated as done. Delete sits behind a dialog that lists what goes.

**The fake has no sign-in flow, only a signed-in account.** `VITE_FAKE_BACKEND` wires `AccountStore` over
`FakeAccountBackend`, which offers no provider until `offerSignIn()` on the console, so the menu offers no
sign-in there. `signIn("Nova")` on the console links it as a finished sign-in would (`completeSignIn`, with the
name as the code), and from a guest the fake grants the refill a second later: the You tab, the username, the color and the clips
switch, and Sign out in the panel undoes it. To try the flow locally, run the backend with `auth.enabled`,
`auth.turnstile.enabled: false`, `auth.signIn.enabled` and any Google
`clientId`, and the dev server with Turnstile's always-passing test sitekey:

```bash
VITE_API_BASE_URL=http://localhost:8080 VITE_TURNSTILE_SITEKEY=1x00000000000000000000AA npm run dev
```

The start leg is real — the button goes to Google, which refuses a made-up
client — and `/auth/callback?code=x&state=y` exercises the callback page's
refusal. A whole sign-in needs a real client registered with
`http://localhost:5173/auth/callback`. To see the signed-in panel without one,
mint a guest and insert a row into `auth.identities` for its account.
