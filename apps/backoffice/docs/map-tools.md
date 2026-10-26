# The map tools

`src/map/`, a page each.

- **Reassign** (`ReassignCountry`): every tile one country holds goes to another. The dry run counts both sides.
  A `from_after` above zero is tiles retaken while it ran: the page says to run it again.
- **Paint** (`PaintRandomTiles`): a flag, a ground to start on (empty: the whole map), a count and a proximity
  (0 scattered, 1 one patch). The dry run draws without painting, so the real run draws again: the question says
  "about".
- **A country is typed by its id** (`CountryField`), and the page names it below ("dz" → Algeria) with
  `Intl.DisplayNames`, plus the game's own ids no standard names (`countries.ts`: England, Scotland, Wales,
  Brittany). It is a hint: the server holds the list of countries and refuses any other.
- **A flag is an emoji** (`flagOf` in `countries.ts`, `Country`), never the game's flag atlas, which is the
  frontend's: the regional indicators of an ISO code, the tag sequences of England, Scotland and Wales, and nothing
  for Brittany, which Unicode has no flag for. Windows draws the two letters instead, so a flag always goes beside
  its country's name.
- **Replay** (`GetReplay`): a span back from an end (empty: now), saved as `replay-<since>-<until>.json`, the
  answer as protobuf JSON, which the frontend's `npm run clip` reads as it reads what `clips fetch` saves. The
  ledger keeps 72 hours. The file is made in memory (`download.ts`, a `blob:` link), which the CSP allows.
