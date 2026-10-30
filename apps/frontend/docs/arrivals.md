# Arrivals

**The backend keeps how each new account came, and only the browser knows it.** So `main.tsx` reads the query of
the first page of the game, keeps it, and sends it with every call that can make an account. The backend decides
what it means: see the backend's [Arrivals](../../backend/docs/arrivals.md).

- **The landing is the first page's `?f=` and `?s=`** (`landingOf` in `src/domain/landing.ts`): a flag the game
  knows (`sharedCountry`, the same read the recruit link switches the flag with) and the tag in lower case. The
  YouTube link is `https://clickplanet.lol/?s=yt`.
- **It is kept from the first page on** (`firstLanding`, `localLandingStore`, key `clickplanet-landing`), an empty
  one too, so a later link never takes its place. The account is made at the first click or at a sign-in, often
  after a reload or the sign-in's round trip to Google, whose callback page has no query of its own.
- **It goes with `CreateSession`** (`SessionClient`'s `landing` option) **and both sign-in completions**
  (`ConnectAccountBackend`). The server keeps it only when that call makes the account, so sending it on every
  call costs nothing.
- **Both parameters are taken out of the address bar** (`withoutLanding`) once read: left there, a copied address
  would make a friend a YouTube arrival, and a reload would undo a flag switched since.
- **The home page carries the query to `/play`** (see [Pages and routes](routes.md)), so `/?s=yt` reaches the game.
