# CrazyGames

**The same build is the CrazyGames page when it runs on `crazygames.clickplanet.lol`**, the host CrazyGames frames, or
when it was built with `VITE_CRAZYGAMES=1`. Never on clickplanet.lol. `ON_CRAZYGAMES` (`src/platform.ts`, from
`onCrazyGames` in `src/domain/crazyGames.ts`) says which, once per page. The shared [CrazyGames](../../../docs/crazygames.md)
says why a host of its own and what the owner sets up.

## What changes there

- **`/play` is the game at once, as a guest.** No donation popup (`DonationModal`), no Buy Me a Coffee, no Share tab,
  no recruit link, and About keeps the community links but not the author's: CrazyGames forbids a link to another
  place to play, and payments outside its own. The Invite tab and the frozen map's Invite copy CrazyGames' own link
  instead (see [Multiplayer](#multiplayer)).
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
- **The CrazyGames username is the name.** The server names a new CrazyGames account after it (the backend's
  [Player](../../backend/docs/player.md)), so friends know each other. `AccountPanel` has no username form for an
  account linked to CrazyGames: the name shows, and only its color may change. The sign-in pitch promises that
  username, not a pick, when `crazygames` is offered.
- **The session is carried in a header** (`carriedFetch`, `localAccountCarrier`): see [Sessions](sessions.md).
- **Every new account is tagged `cg`** (`crazyGamesLanding`), whatever the link said: see [Arrivals](arrivals.md).
- **CrazyGames' settings win.** `useCrazyGames` follows `disableChat` (no chat panel, no chat tab) and `muteAudio`
  (every sound and the anthem silent). `useGameplay` tells `loadingStop` and `gameplayStart` once the globe is on
  screen; the game never pauses, so there is no `gameplayStop`.
- **No ad.** The owner chose none: no ad call, no ad button, nothing from CrazyGames' ad module.
- **The words of CrazyGames' list are masked** in the chat and in every player name: see [The word filter](#the-word-filter).

## The word filter

CrazyGames asks every game with a chat for a profanity filter, and gives a list
([its requirement](https://docs.crazygames.com/requirements/multiplayer/)). **The page masks what it shows; it never
changes what is sent or kept.** The server, the stored messages and clickplanet.lol see the text as it was written.

- **The list is fetched at build time, never kept in the repo.** `npm run crazygames:words`
  (`scripts/fetchCrazyGamesWords.mjs`) reads CrazyGames' sheet as CSV, keeps the first column, and writes
  `public/crazygames/words.txt`, which git ignores. The sheet states no licence, and the repo is public. `npm run build`
  runs it first, and fails when the sheet does not answer CSV or has fewer than 1,000 words, so a build never ships
  without the list.
- **The page loads it before it draws anything** (`crazyGamesWords`, `src/backends/crazyGamesWords.ts`), and only
  there: clickplanet.lol gets `NO_FILTER`. A list that does not load, or that comes back as the home page (the
  single-page fallback answers a missing file with it, as 200), is `EVERY_WORD_MASKED`: nothing unfiltered is shown.
- **Whole words and phrases, any case**, as CrazyGames' sheet asks: "ass" is masked, "assassin" is not
  (`wordFilterOf`, `src/domain/wordFilter.ts`). A word keeps its first and last letter, and the ones between become
  `*`: `f**k`, `a*s`; a word of two keeps its first. The punctuation around a word is left out of the match and kept,
  but an entry with symbols (`a$$`) is matched as it is first. A word inside a link is not masked, since the link is
  one word.
- **Every place that shows a message or a player name asks `useWordFilter()`** (`src/app/crazygames/wordFilter.ts`, a
  context `App` provides): the chat's messages, quotes, peek and toast, the mute line, who reacted, the composer's
  name, the online list, the player, holder and territory cards, the standings, the battle debrief, the report forms
  and the account's own name. A new place that shows one does the same. A name CrazyGames gives is a player name like
  any other.

## Multiplayer

CrazyGames lists the game as online multiplayer, so it must report a room, take invites, and copy an invite link.
Instant multiplayer needs nothing: there is no lobby, `/play` is the shared world at once.

- **The whole world is one room** (`WORLD_ROOM`, `"world"`), always joinable. `useRoom` reports it with
  `updateRoom` once the globe is on screen, and again at each change of the flag shown; the invite parameters are
  that flag, `{f: "fr"}` (`inviteParamsOf`). This drives Join in a friend's list on CrazyGames. The report waits
  for the flag to hold still for a second (`ROOM_SETTLE_MS`): the SDK refuses a second report within 250 ms, and
  the flag is often set back at load. The page never calls `leftRoom()`: a player leaves the world only by
  closing the page.
- **An invite gives its flag to a player who plays for none** (`useInvites`): the invite the page opened from
  (`inviteParams`, told once) and each one taken while it plays (`addJoinRoomListener`), with no reload. A
  player who holds a flag keeps it ([The flag played for](flag.md)): only the flag shown changes, never a
  pledge. A flag that is not held yet at the invite is set back when the held one loads.
- **Invite friends** copies `inviteLink({f: <flag shown>})`, a link to the game on CrazyGames (`InviteFriends`,
  `InviteButton`). The Invite tab of the menu and of the phone's tab bar takes Share's place, and the frozen
  map's Invite copies the same link. `Menu`, `TabBar` and `ClickBudgetMeter` show it when they get `invite`,
  which only the CrazyGames page has.

## The SDK (`src/backends/crazyGamesSdk.ts`)

`connectCrazyGames` loads `https://sdk.crazygames.com/crazygames-sdk-v3.js`, awaits `init()`, and answers at once
with a `CrazyGamesPage` (`src/backends/crazyGames.ts`): a call made before the SDK is ready runs once it is. **The game
never waits on it.** An SDK that does not load (an adblocker) or runs off CrazyGames (`environment` `disabled`) is
a guest: no token, a prompt that answers false, the default settings.

## Trying it

`npm run crazygames:words` once, then `VITE_CRAZYGAMES=1 VITE_FAKE_BACKEND=1 npm run dev`: the SDK runs in its local
mode on `localhost`, with a demo user.
`offerSignIn()` on the console offers sign-in and signs in as that user. The SDK's query parameters change what it
answers (`?user_response=logged_out`, `?disableChat=true`, `?muteAudio=true`). An invite is
`/play?czy_invite=true&czy_invite_params=f%3Dbr`, the form of `inviteLink` (a plain `&f=br` is a share link:
`main.tsx` removes it before the SDK reads it). The local SDK logs each room report and drops the join listener,
so an invite taken while playing is only in the tests.
