# Running it locally

The page needs a backend with auth on and an admin listener. In a copy of `apps/backend/cmd/api/example.yaml`:
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
