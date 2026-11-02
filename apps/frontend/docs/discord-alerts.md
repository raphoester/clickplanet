# Discord pings

**A player can get pinged on Discord.** One press joins them to the game's Discord server with the role of the flag
they play for, and Discord pings that role when the flag is attacked and when a battle starts. The server does the
pinging (the backend's `alerts` module); the page only offers the link and finishes it.

- `backends/alerts.ts` — the contract: `AlertsBackend` (`discordLink`, `startDiscordLink`, `completeDiscordLink`),
  `DiscordLink` (`offered`, `linked`) and `AlertsError`, whose `failure` says why the server said no.
- `backends/alertsBackend.ts` — `ConnectAlertsBackend`, over `alerts.v1.AlertsService`. It sends the click token
  with every call and sends a call refused `unauthenticated` once more with a fresh session, as the player backend
  does. **Only the read is retried**: a retried `CompleteDiscordLink` would spend a code that is good once.
- `app/alerts/discordAlerts.ts` — `DiscordAlerts`: `read`, `leave` (start, remember the trip, go to Discord) and
  `complete`. `main.tsx` builds it with `window.location.assign` and `rememberDiscordLink`.
- `backends/fakeAlertsBackend.ts` — in fake mode the link is offered and not made; Start comes straight back to
  `/auth/callback` with a fake code, and Complete makes it.

## What the player sees

**Nothing unless the server offers it and this account has not taken it.** `useDiscordAlerts` (in `Viewer`) reads
`GetDiscordLink` at load and again when the account changes. A 404, `unimplemented` or any other failure reads as
not offered, as `GetSignInOptions` is read.

- **The chat's Discord bar** (`DiscordBar`): "Get pinged" with the Discord logo takes the place of "Join Discord",
  named "Get pinged on Discord". A linked player keeps "Join Discord". A refused start puts the reason on the
  button ("Wait a minute", "Try again"), and a press tries again.
- **A moment when the flag is under attack** (`UnderAttack`, `useUnderAttack`): "France is under attack" and the
  same button, for 8 s, in the Moments zone. **Once a page**, the first time the flag the player took a tile for in
  this page loses ground on the board (its badge in `tileDeltas` goes below 0). A player who has not taken a tile
  sees nothing: a first visit would see it before it played.
- **Not on CrazyGames**: the trip leaves its frame and comes back on clickplanet.lol, where its carried session is
  not.

## The trip

**Discord sends the browser back to sign-in's redirect URI**, `/auth/callback?code=…&state=…`: the owner registers
no new URL. So before it leaves, the page remembers that this trip is a Discord link (`rememberDiscordLink`,
`clickplanet-callback-trip` in session storage), and `rememberSignIn` forgets it. On `/auth/callback`, `SignInGate`
renders `DiscordLinkCallback` instead of `SignInCallback` when that is what was remembered. The page keeps the
callback page's protections against a leaked code (see [Sign-in](sign-in.md)): the same path, the same
`no-referrer`, the query taken out before anything renders.

`DiscordLinkCallback` calls `CompleteDiscordLink` **once** (a ref guards it, as StrictMode runs the effect twice),
on the sign-in callback's card:

- **Done**: "You are in!", "Open Discord" (the server's page, in a new tab) and "Back to the game".
- **Refused**: "Not linked" and one line. Every failure but `off` has "Try again", which starts a new trip.

| Code | `failure` | Line |
|---|---|---|
| `failed_precondition` | `startAgain` | This link is too old, or it started in a different browser. Start again. |
| `permission_denied` | `refused` | Discord did not accept it. Try again. |
| `unauthenticated` | `notSignedIn` | We could not check who you are. Try again. |
| `resource_exhausted` | `tooManyTries` | Too many tries from your network. Wait one minute, then try again. |
| `unimplemented` | `off` | Discord pings are not available now. |
| anything else | `failed` | Something went wrong. Try again. |

- **Said no on Discord** (`?error=`): "You did not link Discord". **A link with a part missing**: "This link is not
  complete". Both offer "Try again" and the way back.
