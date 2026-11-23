# The list of flags

**The flags the game ships with are in the build. The ones opened since come from the server.**
`static/countries/countries.json` is the built-in list: the codes, their order, their pictures in the atlas and their
names in every language. The backend's flags module owns the whole catalog: `flags.v1.FlagService/ListFlags` lists the
built-in flags first, then the opened ones, each with a name in English and whether it is closed. An opened flag's code
is `r-` and ten letters or digits (`isOpenedCode`). None is opened yet (2026-11-23), so no player sees a change.

## The catalog

- **`FlagCatalog`** (`domain/flags.ts`) is the built-in flags merged with the server's list. `knows`, `playable`,
  `playableCodes` (the picker's) and `openedName`. A built-in flag keeps the page's own names, whatever the list calls
  it.
- **A closed flag stays known**: history and the boards still name it. But no player may play for it, so it is not in
  `playableCodes`, and the backend refuses a click or a pledge for it as for an unknown flag.
- **The store** (`useFlags`, `app/flags/flagsStore.ts`) holds the catalog. `useFlagsFeed`, which `PlayScreen` calls,
  serves it `Backends.flags` (`ConnectFlagsBackend`, or `FakeFlagsBackend` in fake mode) and listens to the flags
  module's stream through it (see [backends.md](backends.md#the-flags-stream)). The archive page serves it alone: it
  reads the list once, with no stream.

## When the page reads the list

- **At load**, once.
- **Each time the flags stream opens**, the first time included: nothing told before it opened is sent again.
- **Not on `flag_changed`**: the stream tells the flag as it is now, and the store merges it with no read. A flag told
  during a read is merged again over that read's answer, which may predate it.
- **When a lookup meets a code the catalog does not know**: the name of an opened code (`translator().country`), a
  `CountryFlag` with no picture, and `knowsFlag`.
- **At most one read is in flight.** A stream opening during a read queues one more read after it, never two. A code missed
  during a read waits for that read.
- **A code reads the list once.** A code still unknown after its read never starts another one. On 2026-10-31 a new
  enum value made old tabs reconnect every 500 ms, so an unknown flag must not loop like that (see
  [backends.md](backends.md)).
- **The load read may come from the browser's cache. Every later read goes past it** (a second client, with
  `cache: "no-cache"`): it reads because the list may have changed.
- **A read that fails keeps the flags the page knows**, and logs. A server with no flags module answers
  `Unimplemented`, which lists none: the built-in flags stay.

## A code the page does not know

Each place that looks up a flag by its code survives a code it does not know.

- **Its name** (`translator().country`): a built-in code reads as before (the catalogue's lines, then Intl). An opened
  code reads as the list's English name, which is not translated. Until the list names it, it reads as its code. The
  translator keeps neither in its cache. "In" an opened flag (`countryIn`) is the flag, then the name. A name already on
  screen shows the code until its component draws again.
- **Its picture** (`CountryFlag`): the atlas's sprite, or a plain grey flag (`.country-flag-blank`, `--text-faint`) for a
  code with no picture. With no code, nothing.
- **The globe**: a flag with no picture wears `BLANK_NUMBER`, a blank rectangle of the atlas (`BLANK_REGION`). It is
  found at load where no picture is, so it is transparent, and the shaders draw it white. The tiles, the flags painted
  on a landmass and the footprints all use it. Before 2026-11-23 such a tile kept the flag it had, and the globe dropped
  a click for such a flag.
- **The share card** draws a grey rectangle in place of the flag. **A rally's pin** keeps its plain cream cloth.
- **The picker and the knock-out's out window** list `playableCodes()`: no closed flag, and each opened flag by its name.
- **The leaderboard and the race** rank every code the server counts, known or not.
- **The held flag** the server tells (`GetAllegiance`) is set as it is, known or not, and so is its chat room.
- **The country kept in storage** may be a built-in code or an opened one, which the page keeps before it reads the list.
- **A share link** (`/r?f=`, see [Sharing](sharing.md)): a built-in flag is chosen before the first render, as before.
  An opened flag waits for the first read, and is chosen if the list says it is open and the player has not changed
  flag in the meantime. Any other code is ignored.
- **The standings** draw the flag of any code but an empty one. **The anthem** has no recording for it, so its play
  button is off.
- **Built-in flags alone**: a flag's old address (`clickplanet.lol/<flag>/<name>`, written into the home page at the
  build) and the clips. An opened flag never had an old address.

## In fake mode

`openFlag("Bavaria")` on the console opens a flag and `closeFlag(code)` closes it. Each tells the store as the flags
stream would.

## Not built yet

An opened flag's picture (the next step), and its name in other languages.
