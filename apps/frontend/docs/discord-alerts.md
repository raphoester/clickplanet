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
- **A moment when the flag is under attack** (`UnderAttack`, `useUnderAttack`): "France is under attack" and "Get
  pinged", the same sign-in, for 8 s, in the Moments zone. **Once a page**, the first time the flag the player took a
  tile for in this page loses ground on the board (its badge in `tileDeltas` goes below 0), and only while `connect` is
  offered. A player who has not taken a tile sees nothing: a first visit would see it before it played.
- **Not on CrazyGames**: the trip leaves its frame and comes back on clickplanet.lol, where its carried session is
  not; and CrazyGames allows no sign-in but its own.

## The trip

**It is the sign-in's**: `/auth/callback`, `SignInCallback`, `CompleteSignIn`, and the game swapped in place (see
[Sign-in](sign-in.md)). The backend joins the server inside `CompleteSignIn`, so the page that comes back reads the
account as linked. A join that failed signs the player in all the same, and the bar offers to join again.

**To try the offer again**, an operator unlinks the account on the backoffice's player page (the Discord tool): the
next load offers "Join Discord" again. A player cannot unlink: leaving the server is how they stop the pings.

**A player who signed in with Discord before the sign-in joined the server** has no link, so the page offers them the
same "Join Discord" and "Get pinged": for an account signed in already it is a link of the identity it holds, which
joins the server like any Discord sign-in. Nothing else could add them: Discord adds a user to a server only with that
user's own consent. The ones in the server already, by the invite, need no press: the backoffice's "Link Discord
members" gives them their role (see the backend's [Discord pings](../../backend/docs/alerts.md#the-players-a-sign-in-never-joined-linkmembers)).
