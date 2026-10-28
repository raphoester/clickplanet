# Running it locally

**To see the page without a backend, `npm run demo`** (`vite --mode demo`, on `:5174`). The mode serves
`src/demo/api.ts` in place of `src/api.ts` (an alias in `vite.config.ts`): every client answers in the page, from
`DemoAdmin` (`src/demo/demoAdmin.ts`), a `FakeAdmin` with a hundred and twenty made-up players, a day and a half of
chat with insults in it, a dozen reports and four rule sets, the same on every load. It keeps what the operator does
(a ban, a mute, a closed report, a saved set) until the page reloads, and answers each call after a quarter of a
second, so the loading states show. `WhoAmI` answers a made-up operator, so there is no sign-in. A production build
never holds it: the alias is the demo mode's alone.

**For the real API**, the page needs a backend with auth on and an admin listener. In a copy of `apps/backend/cmd/api/example.yaml`:
`httpServer.adminBindAddress: 127.0.0.1:8083`, `auth.admin.origin: http://localhost:5174`, `auth.enabled: true` and
an `auth.secret` (`openssl rand -hex 32`). Postgres on `localhost:5432`. Then
`VITE_API_BASE_URL=http://localhost:8083 npm run dev`.

- **The boot needs `auth.admin.google.clientId` and `clientSecret`** once `auth.admin.origin` is set. Made-up
  values boot it, and a session made by hand (below) works with them. **A whole Google sign-in** needs a real
  Google client that lists `http://localhost:5174/auth/callback`, and a Google identity on an account flagged
  operator.
- **Without Google**, open a session by hand. The table keeps the SHA-256 of the cookie's value, so pick a value,
  insert its hash, and set the cookie on `http://localhost:5174` (a cookie ignores the port, so it reaches `:8083`;
  the backend does not see that it is not `HttpOnly`):

  ```sql
  UPDATE auth.accounts SET operator = true WHERE id = '<account id>';
  INSERT INTO auth.admin_sessions VALUES (sha256('local-test'), '<account id>', now(), now() + interval '12 hours');
  ```

  ```js
  document.cookie = "cp_admin=local-test; path=/"
  ```

  Any account row will do, a guest's included (`curl` a `CreateSession` against `:8080`): only the Google
  sign-in checks the identity, and this skips it.
