# CrazyGames

**The same build is the CrazyGames page when it runs on `crazygames.clickplanet.lol`**, the host CrazyGames frames, or
when it was built with `VITE_CRAZYGAMES=1`. Never on clickplanet.lol. `ON_CRAZYGAMES` (`src/platform.ts`, from
`onCrazyGames` in `src/domain/crazyGames.ts`) says which, once per page. The shared [CrazyGames](../../../docs/crazygames.md)
says why a host of its own and what the owner sets up.

## What changes there

- **`/play` is the game at once, as a guest.** No donation popup (`DonationModal`), no Buy Me a Coffee, no Share tab,
  no Invite on a frozen map, and About keeps the community links but not the author's: CrazyGames forbids a
  link to another place to play, and payments outside its own.
- **The only sign-in is CrazyGames'.** `AccountStore` gets the SDK's user (`crazyGames` option) and then offers
  `crazygames` alone; the site never offers it. The button opens CrazyGames' own prompt (`showAuthPrompt`), never
  on its own. See [Sign-in](sign-in.md).
- **The page plays as whoever CrazyGames says.** `followCrazyGames` runs at every start and at every sign-in
  CrazyGames tells (`addAuthListener`): a signed-in player gets its token (`getUserToken`, `IN_GAME_AUTH`) and
  `CompleteCrazyGamesSignIn`; an account linked to CrazyGames whose player is no longer signed in there (another
  player on the device, or a sign-out on CrazyGames) is signed out; a guest stays a guest. Two at once share one
  call.
- **No sign-out and no link.** `AccountPanel` hides Sign out and Sign out everywhere for an account linked to
  CrazyGames; Delete account stays.
- **The session is carried in a header** (`carriedFetch`, `localAccountCarrier`): see [Sessions](sessions.md).
- **Every new account is tagged `cg`** (`crazyGamesLanding`), whatever the link said: see [Arrivals](arrivals.md).
- **CrazyGames' settings win.** `useCrazyGames` follows `disableChat` (no chat panel, no chat tab) and `muteAudio`
  (every sound and the anthem silent), and silences the game while an ad plays. `useGameplay` tells `loadingStop`
  and `gameplayStart` once the globe is on screen.

## The SDK (`src/backends/crazyGamesSdk.ts`)

`connectCrazyGames` loads `https://sdk.crazygames.com/crazygames-sdk-v3.js`, awaits `init()`, and answers at once
with a `CrazyGamesPage` (`src/backends/crazyGames.ts`): a call made before the SDK is ready runs once it is. **The game
never waits on it.** An SDK that does not load (an adblocker) or runs off CrazyGames (`environment` `disabled`) is
a guest with no ad: no token, a prompt that answers false, ads blocked, the default settings.

## The ad refill

**At the empty bank, every player is offered a rewarded ad for a full bank**, guest or signed in. The server
fills the bank and bounds what a client can get without an ad: see the backend's
[A rewarded ad fills the bank](../../backend/docs/ad-refill.md).

- **`useAdRefill`** asks the SDK about an adblocker, then the server when the bank may have its next one
  (`GetAdRefill`): `ready`, `waiting` until a time, or `blocked`. `off` while the rule is off (`BonusRules.adRefill`)
  or off CrazyGames. `watch` plays the ad (`requestAd("rewarded")`) and, only once it played to its end, asks the
  server (`ClaimAdRefill`). An ad that did not play is never claimed: `unavailable` (no ad now, try later),
  `blocked`, or `off` (`adsDisabled`, as in the Basic Launch, so the offer goes for the page).
- **The dry tap opens it** (`AdRefillModal`), where the site asks a guest to sign in: the same `useSignInNudge`,
  once a page, never over a quiz, not while a refill is held. "Watch an ad: full bank" and "No thanks" look the same.
  It cannot be closed while the ad plays, and says the bank is full after.
- **It stays in one place**: the clicks panel offers it while the bank is not full, and says when the next one is,
  or that ads are blocked.
- **The game is silent while the ad plays** (from `adStarted`, not from the request) and tells `gameplayStop`, then
  `gameplayStart` after.

## Trying it

`VITE_CRAZYGAMES=1 VITE_FAKE_BACKEND=1 npm run dev`: the SDK runs in its local mode on `localhost`, with a demo user
and a text overlay for an ad. Its query parameters change what it answers (`?user_response=logged_out`,
`?disableChat=true`, `?muteAudio=true`). The fake gives one ad refill an hour.
