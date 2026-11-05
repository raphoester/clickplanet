# Connecting Claude

**`/connect` is where the operator lets an app in**: Claude, as a custom connector (the backend's
[Claude as a connector](../../backend/docs/connector.md)). It is not in the menu. The backend's authorize endpoint
sends the browser there, with the app's request sealed in `?request=`.

- **`main.tsx` keeps the request for the tab** (`connect/pending.ts`, `sessionStorage`) and takes it out of the
  address bar, as it does a sign-in code. An operator who is not signed in signs in with Google first, comes back on
  `/auth/callback`, and is asked then. If storage is blocked, the request is lost, and the operator connects again
  from Claude.
- **`ConnectPage` asks `DescribeConnection`** and shows the app's name and the host it will be sent back to, with
  Allow and Deny. **Allow** calls `ApproveConnection` and sends the browser to the address it answers: the app's
  callback with a code. **Deny** sends it to the app's callback with `access_denied`. Both forget the request. A
  request past its ten minutes (`failed_precondition`), or one the server did not seal, says to connect again and
  offers the backoffice instead.
- **Only the page's session can let an app in**: the gate refuses both calls to a bearer token, so a connected app
  never connects another.
