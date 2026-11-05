# A server restart reloads the page

**The game page reloads itself when the backend it talks to restarts.** The frontend (Cloudflare Pages) and the
backend deploy apart, and a page stays open for hours. Before this, every change of the wire had to keep the old
shape for the tabs opened before the deploy: the old field deprecated, both paths served, the old one cleared in a
second PR. Now a page that sees a new server reloads, and gets the bundle built for it. So a change of the wire can be
a **hard switch**: the new paths and messages only, in one PR.

## The server

- **`cpbootstrap` draws a boot id at each start** (`cpsecrets.RandomHex`, before any module is built), and
  `cphttpserver.NewBootMiddleware` puts it on every answer of the public router, in the `X-Server-Boot` header.
  `TestEachRunOfTheProcessAnswersUnderABootOfItsOwn` pins that one run keeps one boot and the next one draws another.
- **CORS must expose it**, or the browser hides it from the page: `NewCorsMiddleware` and Caddy's
  `Access-Control-Expose-Headers` (`deploy/vps/caddy/Caddyfile`) both list it beside `X-Account-Session`.
- **Only the public router.** The console, admin and internal listeners carry no boot: the backoffice does not reload,
  so a hard switch of an operator's call needs the operator to reload the admin page.

## The page

- **`restartInterceptor`** (`apps/frontend/src/backends/restarts.ts`) is in the one interceptor list `main.tsx` gives
  every Connect client. `Restarts` keeps the first boot it reads. A second, different one reloads the page, once
  (`location.reload()`).
- **It reads the answers and the refusals** (a `ConnectError` carries the headers in its `metadata`), and a stream
  as it opens. The drain ends every stream at a restart, and the planet stream opens again 500 ms later, so a page
  reloads within a second of the new server answering.
- **It skips a `NO_SIDE_EFFECTS` unary call.** That call may go as a GET, and a GET may come from the browser's
  cache, under the boot of a server that is gone. Every stream and every other call is a POST.
- **An answer with no boot changes nothing**: the fake backend (`VITE_FAKE_BACKEND=1`), or a route that never reaches
  the backend.

## What a player sees

At each deploy, and after a crash that restarts the container, the page loads again: the globe loads again, a chat
message being typed is lost, and a click in flight may be lost. The click token and the identity stay in the browser,
so the player plays on as before.

## The order of the deploys

**The new frontend must be live before the backend restarts.** One merge starts both: Pages builds the frontend, and
the `deploy backend` workflow builds the image and restarts the container (about 3 minutes, sometimes much more).

- **Frontend first** is the normal case. A page loaded between the two is the new bundle on the old server: what it
  calls new fails until the backend restarts, and the restart reloads it.
- **Backend first** reloads every page onto the old bundle, and nothing reloads it again until the next restart.
  When a merge with a hard switch deployed in that order, run `deploy backend` again (`workflow_dispatch`) once Pages
  is live: the second restart reloads the pages onto the new bundle.
