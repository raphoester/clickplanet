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
  on its own, and the empty bank asks nothing (`useSignInNudge` is off): CrazyGames forbids a sign-in that pops up
  or is the main call to action. See [Sign-in](sign-in.md).
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
  (every sound and the anthem silent). `useGameplay` tells `loadingStop` and `gameplayStart` once the globe is on
  screen; the game never pauses, so there is no `gameplayStop`.
- **No ad.** The owner chose none: no ad call, no ad button, nothing from CrazyGames' ad module.

## The SDK (`src/backends/crazyGamesSdk.ts`)

`connectCrazyGames` loads `https://sdk.crazygames.com/crazygames-sdk-v3.js`, awaits `init()`, and answers at once
with a `CrazyGamesPage` (`src/backends/crazyGames.ts`): a call made before the SDK is ready runs once it is. **The game
never waits on it.** An SDK that does not load (an adblocker) or runs off CrazyGames (`environment` `disabled`) is
a guest: no token, a prompt that answers false, the default settings.

## Trying it

`VITE_CRAZYGAMES=1 VITE_FAKE_BACKEND=1 npm run dev`: the SDK runs in its local mode on `localhost`, with a demo user.
`offerSignIn()` on the console offers sign-in and signs in as that user. The SDK's query parameters change what it
answers (`?user_response=logged_out`, `?disableChat=true`, `?muteAudio=true`).
