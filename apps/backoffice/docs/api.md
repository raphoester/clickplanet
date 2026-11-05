# How it reaches the API

- **Every call goes to `https://api.clickplanet.lol/<package>.v1.AdminService/<Method>`** (`src/api.ts`).
  Caddy sends those paths to the backend's admin listener (the backend's [Operator tools](../../backend/docs/admin.md)).
  `VITE_API_BASE_URL` points it elsewhere; it is a build variable.
- **Calls go in binary protobuf, not JSON** (`useBinaryFormat`): writing a page of the ledger as JSON took the
  server 8 times as long. An error still comes back as JSON, and a replay is saved as JSON on this side
  (`toJsonString`).
- **The page signs in on its own, with Google only** (`auth.v1.AdminService`, the backend's
  [Operators](../../backend/docs/auth.md#operators)). Its session is the `cp_admin` cookie: 12 hours, never extended, `SameSite=Strict`, host-only on
  `api.clickplanet.lol`. It travels from `backoffice.clickplanet.lol` because both are one site, and the transport sends
  `credentials: "include"`. **The game's `cp_sid` never opens this page**, and signing out of one leaves the other.
- **The sign-in** (`src/signIn/`): "Sign in with Google" calls `StartSignIn` and sends the browser to the
  `authorization_url` it answers. Google comes back to `/auth/callback` (the backend's `auth.admin.redirectUrl`).
  `main.tsx` reads the code and the state there (`callbackOf`) and **takes them out of the address bar before
  anything renders**: a code is good once. `SignInCallback` calls `CompleteSignIn` once (a ref guards it, since
  StrictMode runs an effect twice), then shows the page. A Google refusal, or a link with a part missing, offers
  the button again. The Worker serves `/auth/callback` through its SPA fallback.
- **Sign out** is a button beside the account id (`SignOut`), and then the page asks `WhoAmI` again.
- **The gate admits a session of an account whose `auth.accounts.operator` is true**, and only from the `Origin`
  `https://backoffice.clickplanet.lol`. The flag is set by hand (`deploy/vps/README.md`, "Make an account an
  operator"); it is read on every call, so unsetting it ends a session already open.
- **The refusals** (`refusal.ts`, `FailureNotice`): `unauthenticated` is "Sign in to continue", `permission_denied`
  is "This Google account is not an operator" (a wrong origin too), `failed_precondition` from `CompleteSignIn` is
  "The sign-in expired. Start again". Each offers the Google button. Anything else is "The call failed", with Try
  again. The server's message is shown under each, for the operator.
- **`auth.v1.AdminService/WhoAmI` says who the page is**: the operator's account id, from the `cp_admin` session.
  `App` asks it first, and shows nothing else until it answers.
- **`useCall`** runs one call, aborts it when the component goes or the call changes, and holds `loading`,
  `done` or `failed`. Its `run` must keep its identity (module scope or `useCallback`).
