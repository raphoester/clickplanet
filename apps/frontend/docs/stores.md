# Stores

**A feature's state lives in its own store, not in `Viewer`.** `Viewer.tsx` is where the features met: 6 hooks on
2026-09-15, 61 on 2026-11-03, the file doubling every two weeks. Each piece of state it held made all of it render
again on a change, and was passed down by hand to the components that showed it.

A store is a [Zustand](https://zustand.docs.pmnd.rs) store, in the feature's folder, named `<feature>Store.ts`. It
exports one hook, `use<Feature>`, made with `create`. It holds the state and the verbs that change it
(`setMapView`), and saves itself where it must (a `subscribe` that writes `localStorage`, in the format the page read
before). For example [`app/settings/displaySettingsStore.ts`](../src/app/settings/displaySettingsStore.ts).

- **The component that shows the state reads the store itself.** `SettingsPlace` reads the display settings, the
  board reads its guide, and the phone's tab bar the chat's unread count. `Viewer` reads a store only for what it
  draws or hands a hook itself, such as the two display fields it gives the globe, and never to pass a value down.
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
- **What a backend sends goes into its store from where its inputs are.** A hook that `Viewer` calls with the
  backend and what it knows links the two (`useQuizMaster`, `useRosterFeed`, `useRaceFeed`): it subscribes, writes
  the store, and puts the store back to its start once it stops, so nothing stale shows after it and StrictMode's
  second run follows one stream. `Viewer` reads back only what it hands the globe or a hook itself, such as the
  roster's count, or the race it ranks the countries by for the anthem. Or the place where a card shows does it, given
  its inputs as props (`PlayerCardPlace`, `HolderCardPlace`, `TerritoryCardPlace`), and shows the card from the store.
- **A test sets a store with `use<Feature>.setState(...)` and puts it back in `afterEach`**, since the store lives as
  long as the module.

The features still held in `Viewer` move to stores the same way.

## A store that reads by itself

**When many components each show one entry of the same backend data, by id, the store reads it, not the
components.** The badges ([Badges](badges.md)) are the one so far: a name anywhere asks for its account's badge.

- **A component asks, and reads a plain value**: `useBadge(accountId)` selects `badges[accountId]`, the store's
  own object, and asks the store in an effect after every render. Asking an entry that is fresh or being read costs
  a map lookup and nothing else.
- **What is asked within a frame goes out in one call**, cut at what the server takes (`MAX_BADGES_ASKED`): one
  timer per frame, never a call per component.
- **No entry is read twice at once**, and **an entry is fresh for a while** (`BADGE_FRESH_MS`), then read again
  the next time it is asked. An entry the server left out is kept as "none", so it is not asked again either.
- **A failed read changes nothing on screen** but puts those entries off (`BADGE_RETRY_MS`): what was shown stays,
  and what never was shows nothing.
- **What the page learns itself is written at once** (`wear`, `setStreak`), and **a read sent before it is not
  kept** for that entry: it would bring back what the page just changed. `expire` is for what the page knows
  changed without knowing how: the entry is read again.
- **The backend is served to the store by a hook** (`useBadgesFeed`, called by `Viewer`): the reads wait for it, and
  the store empties once it stops, as any store fed from where its inputs are.
- **The bookkeeping is not state**: what is being read, asked, or due again lives in the store's closure, so it
  never renders anything. A test puts the store back with `serve(undefined)`, which empties both.
