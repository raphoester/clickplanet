# CrazyGames

**ClickPlanet is submitted to CrazyGames as an iframe game: `https://crazygames.clickplanet.lol/play`.** The same
frontend build, the same API, the same map and players as clickplanet.lol.

## Why a host of its own

- **The page must know it is on CrazyGames**, and clickplanet.lol must never think it is: there, Google, Discord and
  email sign-in, the donation link and the share links are forbidden. The host says so before any script runs, with
  no query a player could type (the frontend's [CrazyGames](../apps/frontend/docs/crazygames.md)).
- **Framed, not uploaded.** An uploaded build would be a second copy to upload again after each frontend change,
  talking to an API that moved on. Framed, it ships with every frontend deploy. CrazyGames frames games from
  independent domains.
- **The API is reached from a cross-site frame**, so its cookie is a third-party one there. The page carries its
  session in a header instead (the backend's [Auth](../apps/backend/docs/auth.md#a-page-that-keeps-no-cookie-internalauthinternalsessionheader)).

## What each part does

- **Frontend**: the CrazyGames page — CrazyGames' sign-in alone, its settings, the gameplay events, and no link to
  another place to play. No ad, by choice.
- **Backend**: `CompleteCrazyGamesSignIn` checks the SDK's token against CrazyGames' keys and links the account
  (`auth.crazyGames.gameId`); arrivals are tagged `crazygames`.
- **Deploy**: `CRAZYGAMES_ORIGIN` in `deploy/vps/env.public` is the second origin Caddy answers CORS for, with
  `X-Account-Session` allowed and exposed; `crazygames.clickplanet.lol` is in `auth.turnstile.hostnames`.

## What the owner sets up

1. In Cloudflare, add `crazygames.clickplanet.lol` as a custom domain of the Workers project `clickplanet`, and add the
   host to the Turnstile widget's hostnames.
2. In the CrazyGames developer portal, submit the iframe URL `https://crazygames.clickplanet.lol/play`, then set the
   game's id as `auth.crazyGames.gameId` in `deploy/vps/backend.yaml`. Until then nobody signs in on CrazyGames:
   every player plays as a guest.
