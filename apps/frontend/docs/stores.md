# Stores

**A feature's state lives in its own store, not in `PlayScreen`.** `PlayScreen.tsx` is where the features met: 6 hooks
on 2026-09-15, 61 on 2026-11-03, the file doubling every two weeks. Each piece of state it held made all of it render
again on a change, and was passed down by hand to the components that showed it.

A store is a [Zustand](https://zustand.docs.pmnd.rs) store, in the feature's folder, named `<feature>Store.ts`. It
exports one hook, `use<Feature>`, made with `create`. It holds the state and the verbs that change it
(`setMapView`), and saves itself where it must (a `subscribe` that writes `localStorage`, in the format the page read
before). For example [`app/settings/displaySettingsStore.ts`](../src/app/settings/displaySettingsStore.ts).

- **The component that shows the state reads the store itself.** `SettingsPlace` reads the display settings, the
  board reads its guide, the phone's tab bar the chat's unread count, `ScheduleChip`, its sheets and `Moments` the
  schedule, and the clicks meter and panel the click budget. `PlayScreen` reads a store only for what it draws or
  hands a hook itself, such as the two display fields it gives the globe, and never to pass a value down. It reads
  plain fields where it can (the budget's sign-in multiplier, freeze and fast start, not the reading), so a store
  that changes often does not render the whole screen.
- **The country played is a store too** (`useCountry`, `countryStore.ts`), read by the status bar, the news line,
  the moments, the clicks panel, the invite sheet and the territory card. It reads the stored country when it is
  made, but `useKeepCountry`, which `PlayScreen` calls, writes it, from the page's first render: the home page and
  the front guide tell a first visit by its absence, so it is not written on import. `main.tsx` chooses the shared
  flag before the first render. The board and the chat still take it as a prop.
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
- **What a backend sends goes into its store from where its inputs are.** A hook that `PlayScreen` calls with the
  backend and what it knows links the two (`useQuizMaster`, `useRosterFeed`, `useRaceFeed`, `useScheduleFeed`,
  `useMissionsFeed`, `useClickBudgetFeed`): it subscribes, writes the store, and puts the store back to its start once it stops, so nothing
  stale shows after it and StrictMode's second run follows one stream. `PlayScreen` reads back only what it hands the
  globe or a hook itself, such as the roster's count, the race it ranks the countries by for the anthem, the turn it
  makes the mood and the globe's halo from, or the missions it draws on the globe. Or the place where a card shows does it, given its inputs as props (`PlayerCardPlace`,
  `HolderCardPlace`, `TerritoryCardPlace`), and shows the card from the store.
- **What the globe tells goes into the store of its feature.** `useGlobe` hands `createGlobe` the verbs of four
  stores, and puts the stores back to their start when it unmounts:
  - `useRefusedClicks` (`clicks/refusedClicksStore.ts`): the count of clicks the bank or a frozen map refused,
    the VPN, ban and session flags, and the flag that locked a click out. `ClickBudgetMeter` and the three modals
    read it; `PlayScreen` reads the count only to hand it to `useSignInNudge`.
  - `useBonus` (`bonus/bonusStore.ts`): the box won, read by `BonusAward`, and what `Inventory` reads: the notice,
    the charges, the rules, the switches and the armed bomb. `PlayScreen` reads the charges and the rules only for
    its hooks and the clicks panel's toll. `MissionsDone` reads the rules for the rewards it names.
  - `useNews` (`news/newsStore.ts`): the news line, below.
  - `useStanding` (`knockout/standingStore.ts`): while the knock-out plays, the capitals' flags and the flags out, which
    the out window, the picker and the Change button read, and the flag whose out window was closed (see
    [The capitals](capitals.md)).

  So `useGlobe` returns only what `PlayScreen` hands back to the globe or lays the screen out with.
- **A test sets a store with `use<Feature>.setState(...)` and puts it back in `afterEach`**, since the store lives as
  long as the module.

The features still held in `PlayScreen` move to stores the same way.

## The news line

**The line at the top shows one news at a time, the newest.** `useNews` keeps the last bomb, fortify and attack, in
the order they were told (`told`, one counter for the three), and `newestNews` is the one shown; once it is done, the
one before it shows again. The globe tells the bombs and the fortifies, and `useUnderAttack` the attack. `News`
(`news/News.tsx`) shows the newest as `BombNews`, `FortifyNews` or `UnderAttack`, the mission's progress when
there is none (see [Missions](missions.md)), and, after the wave of this browser's first fortify, `FortifyCard`.

## A store that reads by itself

**When many components each show one entry of the same backend data, by id, the store reads it, not the
components.** The badges ([Badges](badges.md)) and the nameplates ([Nameplates](nameplates.md)) are the two so far: a
name anywhere asks for its account's nameplate and badge. `useTeaserCards` (`app/battle/teaserCardsStore.ts`) is a
smaller one: a battle's cards, asked by its key by the teaser and the "See the rules" buttons, read once a page, with
no batch and no freshness (see [Battles](battle.md#the-rules-asked-for)).

- **A component asks, and reads a plain value**: `useBadge(accountId)` selects `badges[accountId]`, the store's
  own object, and asks the store in an effect after every render. Asking an entry that is fresh or being read costs
  a map lookup and nothing else.
- **What is asked within a frame goes out in one call**, cut at what the server takes (`MAX_BADGES_ASKED`): one
  timer per frame, never a call per component.
- **No entry is read twice at once**, and **an entry is fresh for a while** (`BADGE_FRESH_MS`), then read again
  the next time it is asked. An entry the server left out is kept as "none", so it is not asked again either.
- **A failed read changes nothing on screen** but puts those entries off (`BADGE_RETRY_MS`): what was shown stays,
  and what never was shows nothing.
- **What the page learns itself is written at once** (`wear`, `setStreak`, `own`), and **a read sent before it is not
  kept** for that entry: it would bring back what the page just changed. `expire` is for what the page knows
  changed without knowing how: the entry is read again.
- **What another feed already says is written at once too** (`learn`): the roster's lines are whole nameplates as they
  are now, so every player online is named with no read.
- **A reader that must not show an entry unread waits for it** (`whenRead`): the chat holds a message until its
  names are read, or a wait has passed, rather than draw a blank name.
- **The backend is served to the store by a hook** (`useBadgesFeed`, called by `PlayScreen`): the reads wait for it, and
  the store empties once it stops, as any store fed from where its inputs are.
- **The bookkeeping is not state**: what is being read, asked, or due again lives in the store's closure, so it
  never renders anything. A test puts the store back with `serve(undefined)`, which empties both.
