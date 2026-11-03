# Discord pings

**A player joins the game's Discord server by signing in with Discord.** One trip: Discord asks the player to let
ClickPlanet sign them in and add them to the server, and the server gives them the role of the flag they play for.
Discord pings that role when the flag is attacked, and everybody in the server when a battle starts. The server does
the joining and the pinging (the backend's `auth` and `alerts` modules); the page only offers it.

There used to be a second Discord trip of its own, "Get pinged", beside "Sign in with Discord": two consents on the
same application, which nobody could tell apart. Now any Discord sign-in, from the menu or from an offer below, is the
connection.

- `backends/alerts.ts` — the contract: `AlertsBackend.discordLink()`, `DiscordLink` (`offered`, `linked`,
  `serverUrl`, the server's `#alerts` page).
- `backends/alertsBackend.ts` — `ConnectAlertsBackend`, over `alerts.v1.AlertsService`. It sends the click token,
  and sends a read refused `unauthenticated` once more with a fresh session, as the player backend does. A 404 or
  `unimplemented` reads as not offered.
- `app/alerts/discordAlerts.ts` — `DiscordAlerts`: `read`, and `connect`, which is `AccountStore.connectDiscord`.
  `main.tsx` builds it, except on CrazyGames.
- `AccountStore.connectDiscord()` — a Discord sign-in: **a link** for an account signed in already (with Google, an
  email or Discord), so the player stays on it; **a sign-in** for a guest, which moves to the account that Discord user
  has, if any. It goes the way every sign-in goes (`leaveFor`, then `/auth/callback`, see [Sign-in](sign-in.md)), and
  throws a refusal for the button that asked.
- `backends/fakeAlertsBackend.ts` — in fake mode the server is offered, and linked once the fake account signed in.
  `offerSignIn()` in the console offers Discord, the offer starts the fake's sign-in, which comes straight back to
  `/auth/callback`, and the bar then says "Open Discord".

## What the player sees

`useDiscordAlerts` (in `Viewer`) reads `GetDiscordLink` at load and again when the account changes, and answers a
`DiscordOffer`: `connect` while the server is offered, the account is not in it and the server offers Discord's
sign-in; `serverUrl` once it is in; neither otherwise. Any failure of the read is no offer.

- **The chat's Discord bar** (`DiscordBar`), on every page with the chat, until the player hides it (it comes back
  the next day):
  - **Not in the server**: "Get pinged when France is attacked" on the box's top line, and "Join Discord"
    under it. The button is the Discord sign-in, named "Join Discord and get pinged when France is attacked". A
    refused start puts the reason on the button ("Wait a minute", "Not available", "Try again"), and a press tries
    again.
  - **In the server**: "Open Discord", the server's `#alerts` page.
  - **Nothing offered** (the alerts off, a failed read, no Discord sign-in, CrazyGames): "Join Discord", the plain
    invite (`DISCORD_INVITE`).
- **A moment when the flag is under attack** (`UnderAttack`, `useUnderAttack`): "France is under attack" and "Go
  there", for 8 s, in the Moments zone, and "Get pinged", the same sign-in, only while `connect` is offered: an account
  in the server, CrazyGames or a page with nothing offered sees no Discord button. **Once a page**, the first time the
  flag the player took a tile for in this page loses ground on the board (its badge in `tileDeltas` goes below 0). A
  player who has not taken a tile sees nothing: a first visit would see it before it played.
- **"Go there" flies the camera to the flag's last tile lost** (`Globe.flyToLastLoss`), down to tiles of 28px as the
  flight to the front does (see [The front, for a first visit](front-guide.md)). The tile is the last one an update took
  from the flag, a bomb cleared or a ghost took (see [Ghosts](ghosts.md)); the globe forgets it when the player plays
  another flag. The moment stays up until its 8 s end, so "Get pinged" is still there when it is offered.
- **Not on CrazyGames**: the trip leaves its frame and comes back on clickplanet.lol, where its carried session is
  not; and CrazyGames allows no sign-in but its own.

## The trip

**It is the sign-in's**: `/auth/callback`, `SignInCallback`, `CompleteSignIn`, and the game swapped in place (see
[Sign-in](sign-in.md)). The backend joins the server inside `CompleteSignIn`, so the page that comes back reads the
account as linked. A join that failed signs the player in all the same, and the bar offers to join again.

**To try the offer again**, an operator unlinks the account on the backoffice's player page (the Discord tool): the
next load offers "Join Discord" again. A player cannot unlink: leaving the server is how they stop the pings.

## A ping's link opens the globe on the attack

An attack's ping links to `/play?lat=40.42&lon=-100.00&zoom=3.0` (see the backend's
[Discord pings](../../backend/docs/alerts.md#the-pings-pings)). `main.tsx` reads it with `linkedShot`
(`domain/linkedShot.ts`) before the first render, and the globe's camera starts on that `Shot`, not at rest
(`createGlobe`'s `linkedShot`). No flight: the player opened the link to see the attack at once.

- **`lat` and `lon` are degrees, `zoom` is the camera's**: 1 at rest, the whole globe's height on the screen, and the
  game's own bounds (0.5 to 50) hold whatever the link asks. A link without one of the three, or with one off the
  globe, opens the game at rest.
- **The direction follows the tile positions**: y is the north pole, and longitude 0 is on -x
  (`scripts/map/coverage.mjs` reads them the same way). `linkedShot.test.ts` pins it.
- **The three are taken out of the address bar**, with `?f=` and `?s=` (see [Sharing the globe](sharing.md)): a link
  copied later is the plain page, not an old attack.
- **The home page sends them on**: `/?lat=…` reaches `/play` with its query, as every link does (see
  [Pages and routes](routes.md)).
