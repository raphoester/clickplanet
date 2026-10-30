# Arrivals

**The backend keeps how each new account came, and only the browser knows it.** So the first page of the site
reads its query and where the player came from, keeps them, and the game sends them with every call that can make
an account. The backend decides what they mean: see the backend's [Arrivals](../../backend/docs/arrivals.md).

- **The landing is the first page's `?s=` and the referrer's host** (`landingOf` in `src/domain/landing.ts`). `rc`
  is a player's link (`shareUrl` adds it), `yt` the YouTube link, `https://clickplanet.lol/?s=yt`. `?f=` is not
  sent: it picks the flag played, and the backend has no use for it.
- **Only the host of `document.referrer`**, never its path or query, which can say who the player is. A page of
  the game's own is no referrer.
- **It is kept from the first page on** (`firstLanding`, `localLandingStore`, key `clickplanet-landing`), an empty
  one too, so a later link never takes its place. **The home page keeps it first** (`home.ts`): on `/play`, the
  referrer of a newcomer is the home page. The account is made at the first click or at a sign-in, often after a
  reload or the sign-in's round trip to Google.
- **It goes with `CreateSession`** (`SessionClient`'s `landing` option) **and both sign-in completions**
  (`ConnectAccountBackend`). The server keeps it only when that call makes the account, so sending it on every
  call costs nothing.
- **`?s=` is taken out of the address bar** with `?f=` (`withoutLanding`, `withoutSharedFlag`): left there, a
  copied address would make a friend a YouTube arrival.
- **The picture's link has no tag** (`shareLabel`): see [Sharing the globe](sharing.md).
- **The CrazyGames page tags every landing `cg`** (`crazyGamesLanding`) and keeps none: its link is not the
  player's to change, and its storage is its own. See [CrazyGames](crazygames.md).
