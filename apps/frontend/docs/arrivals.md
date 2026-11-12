# Arrivals

**The backend keeps how each new account came, and only the browser knows it.** So the first page of the site
reads its query and where the player came from, keeps them, and the game sends them with every call that can make
an account. The backend decides what they mean: see the backend's [Arrivals](../../backend/docs/arrivals.md).

- **The landing is the first page's `?s=` and the referrer's host** (`landingOf` in `src/domain/landing.ts`). `rc`
  is a player's link, `yt` the YouTube link, `https://clickplanet.lol/?s=yt`. `?f=` is not sent: it picks the flag
  played, and the backend has no use for it.
- **A flag's address is a player's link.** `clickplanet.lol/<flag>` and `clickplanet.lol/<flag>/<name>`, which every
  share now carries (see [Sharing the globe](sharing.md)), are opened by the home page's first script as
  `/play?f=<flag>&s=rc`, and `&r=<name>` when the address has a name (see [Pages and routes](routes.md)). An old link
  with `?f=` and `?s=rc` still counts; one with `?f=` alone is no recruit, as before.
- **The recruiter is an account, read from the name** (`recruiterNameOf`, then `firstLanding`). The address writes
  each space of a username as `-`, which no username holds, so the page turns each `-` back into a space, and asks
  player's `GetPlayerByName` which account holds that name (`nameHolder` in `src/backends/playerBackend.ts`; no session,
  since the landing goes with the call that mints the first one). The landing keeps that account as
  `recruiterAccountId`, and the backend keeps it with the arrival. A name nobody holds, a guest's address (no name)
  and a read that fails keep none; the game still opens on the flag.
- **Only the host of `document.referrer`**, never its path or query, which can say who the player is. A page of
  the game's own is no referrer. **A flag's address hands its referrer to the game**: its first script moves to
  `/play` with `location.replace`, after which `document.referrer` is the address itself, so it keeps the page before
  in session storage (`clickplanet-referrer`), and the game reads it once (`handedReferrer` in
  `src/backends/landingStore.ts`).
- **It is kept from the first page on** (`firstLanding`, `localLandingStore`, key `clickplanet-landing`), an empty
  one too, so a later link never takes its place. **The home page keeps it first** (`home.ts`): on `/play`, the
  referrer of a newcomer is the home page. The account is made at the first click or at a sign-in, often after a
  reload or the sign-in's round trip to Google. The landing is written before the name is read, and again with the
  recruiter once it is, so a page closed in between still keeps the recruit.
- **The game reads the landing as a promise** that settles once the name is read (or at once): it goes with
  `CreateSession` (`SessionClient`'s `landing` option) **and both sign-in completions** (`ConnectAccountBackend`),
  each of which waits for it. The server keeps it only when that call makes the account, so sending it on every call
  costs nothing. A browser that came by a player's address with a name holds the recruiter's account in
  `clickplanet-landing` from then on.
- **The newcomer is told who called it, once** ("kutza called you to fight for Poland", on the news line), by the
  name clips' `GetShownNames` answers for the recruiter's account; none when it shows none. See
  [Player videos](player-videos.md#welcome-line).
- **`?s=` and `?r=` are taken out of the address bar** with `?f=` (`withoutLanding`, `withoutSharedFlag`): left
  there, a copied address would make a friend a YouTube arrival, or the recruit of another player.
- **The CrazyGames page tags every landing `cg`** (`crazyGamesLanding`) and keeps none, with no recruiter: its link
  is not the player's to change, its storage is its own, and its first script never reads a flag's address. See
  [CrazyGames](crazygames.md).
