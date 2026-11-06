# Architecture

**Multiplayer globe conquest game** — users click tiles on a 3D globe to claim
them for a country.

The app is **plain Three.js driven imperatively**, not React Three Fiber. R3F is
not a dependency and should not be reintroduced without a reason: scene setup
here is a state machine with an order to it, and the resources it allocates have
to be released by hand.

The layering exists to keep that imperative part as small as possible:

```
domain/    no GPU, no DOM, no React — plain functions and data
backends/  the wire protocol
app/viewer/ WebGL, and the React hook that owns its lifecycle
app/       components
```

## The backends reach the page as one object

**`main.tsx` builds one `Backends` and `App` hands it to `Viewer` whole.** `Backends` is declared in
`Viewer.tsx`, the one place that reads it. Each branch of `main.tsx`, the fake and the real one, fills it, and one
`start` renders the page with it. A new backend is a field of `Backends` and a line in each branch: `App` does not
change.

- **A field is always there**, so `Viewer` does not check for it. Only three can be `undefined`: `crazyGames` off
  the CrazyGames page, `alerts` on it (no Discord sign-in there), and `gifs` on it or with no KLIPY key.
- **`Viewer` still gives each feature its backend as a prop.** The components and hooks take what they use as an
  input, and their tests give them a fake the same way.
