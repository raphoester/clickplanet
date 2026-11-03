# Stores

**A feature's state lives in its own store, not in `Viewer`.** `Viewer.tsx` is where the features met: 6 hooks on
2026-09-15, 61 on 2026-11-03, the file doubling every two weeks. Each piece of state it held made all of it render
again on a change, and was passed down by hand to the components that showed it.

A store is a [Zustand](https://zustand.docs.pmnd.rs) store, in the feature's folder, named `<feature>Store.ts`. It
exports one hook, `use<Feature>`, made with `create`. It holds the state and the verbs that change it
(`setMapView`), and saves itself where it must (a `subscribe` that writes `localStorage`, in the format the page read
before). For example [`app/settings/displaySettingsStore.ts`](../src/app/settings/displaySettingsStore.ts).

- **The component that shows the state reads the store itself.** `SettingsPlace` reads the display settings, and
  the board reads its guide. `Viewer` reads a store only for what it draws or hands a hook itself, such as the two
  display fields it gives the globe, and never to pass a value down.
- **A selector returns a field or a plain value**: `useDisplaySettings((settings) => settings.mapView)`. A selector
  that builds a new object or array on each call makes React render without end; build it in the component with
  `useMemo`, or wrap the selector in Zustand's `useShallow`.
- **Outside React**, `use<Feature>.getState()` reads and `use<Feature>.subscribe` follows.
- **The tiles never go in a store.** The map, its 100 ms batches and the counts stay in `MapTiles` and go to the
  globe without React (see [Data flow](data-flow.md)); React gets samples of them. A store is for what a person
  reads on the screen.
- **A backend that already holds and tells a state is read as it is.** `AccountStore` and `ConnectionHealth`
  subscribe and notify by hand, which is what a store does; a component reads them through their hooks, `useAccount`
  and `useConnection`.
- **A test sets a store with `use<Feature>.setState(...)` and puts it back in `afterEach`**, since the store lives as
  long as the module.

The features still held in `Viewer` move to stores the same way.
