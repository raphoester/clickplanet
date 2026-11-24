# Boards on the page

`src/boards/`. **A tile id is a place on its board alone** (see the root's [Boards](../../../docs/boards.md)): every
answer that names tiles names their board, and the page places them on that board, never on one it assumed.

## Every board, by name

- **`npm run map`** (`scripts/syncMap.mjs`) copies every board in `/map/boards` into `src/gen/map/<name>/`, its manifest
  and its runs, and writes `src/gen/map/boards.ts`: `BOARDS`, each board's manifest and the address Vite serves its
  runs at (`?url`, a file under `/assets/` each, 121 KB for `earth-1`, 282 KB for `earth-2`, 30 KB for `siege-1`). A
  directory whose manifest names another board stops it. Three exist: `earth-1`, and `earth-2` and `siege-1`, which
  nothing plays yet.
- **`boards.ts`**: `BOARDS`, a `Map` by name of `{board, runsUrl}`, and `boardOf(name)`, which refuses a name the page
  does not carry (`earth-9 is not a board this page carries (it has earth-1, earth-2, siege-1)`) and an answer that names none (`""`). A
  page shows the refusal where the tiles would be drawn, and the rest of what it read.
- **`places.ts`**: `loadPlaces(name)` reads a board's runs the first time an answer names it, rebuilds its positions
  and keeps them for the page load, one promise a board (a failed read is forgotten, so Try again reads it again).
  `Places` carries its `board` (the manifest) beside each tile's latitude and longitude, so whatever reads them knows
  the board's spacing. Each tile's latitude and longitude are read as the map's generator reads a point: `asin(y)` and
  `atan2(z, -x)`.
- **`lattice.ts`** is the backoffice's own copy of the game page's lattice: it walks the lattice in the board's order
  (three's walk for `earth-1`, canonical ids for `earth-2` and `siege-1`) and keeps the points the runs hold, to the bit,
  about 0.1 s for `earth-1`, on the page itself. A warped and rotated board's point is its barycentric weights warped
  and summed over the manifest's rotated corners, in the game page's order of operations (see the root's
  [Boards](../../../docs/boards.md#the-geometry)); `boards.test.ts` rebuilds both new boards to their hashes. A rebuild
  whose hash differs from the manifest's `positionsHash` is logged with the board's name and kept. An order but `three`
  or `canonical`, or corners that are not 36 numbers, are refused. The CSP's `connect-src` holds `'self'` for the runs.

**Who reads a board:**

| page | the board it places tiles on |
| --- | --- |
| Home, the heatmap ([home.md](home.md)) | `GetLedgerResponse.board`, the board the world plays |
| Graffiti ([graffiti.md](graffiti.md)) | each `Graffiti.board`, each board read once |
| Flag submissions ([flag-submissions.md](flag-submissions.md)) | `GetTerritoriesResponse.board`, the board the world plays: a pin is in degrees, on no board |
| Download a replay ([map-tools.md](map-tools.md)) | none: it draws nothing, and the file keeps `eras`, each era's board |
| Game rules ([ops.md](ops.md)) | none: it lists `BOARDS`' names for `arena.board` |

No page of the backoffice reads `GetMapHistory` (`TileHits.era`) or the season replay.

## Sizes read the board

`scale.ts`, `scaleOf(board)`, the rule the backend and the game page apply too: a number tuned on `earth-1` reads the
board's spacing instead.

- `lengthRatio` = `board.spacing / EARTH_1_SPACING` (`0.004063093462255102`), `densityRatio` =
  `(EARTH_1_SPACING / board.spacing)²`;
- a count of tiles `n` → `max(1, round(n × densityRatio))`; a length `x` → `x × lengthRatio`; a zoom `z` →
  `z / lengthRatio`.

On `earth-1` both ratios are 1 and every number comes out as it was, to the bit (`scale.test.ts`). The backoffice's
numbers tuned on `earth-1`, and what each became:

| number | where | `earth-1` | now |
| --- | --- | --- | --- |
| a tile's width on the heatmap | `home/heat.ts` `TILE_DEGREES` | 0.3° | `length(0.3)`, 2.9° on `fixture-1`; a tile also covers the pixel rows within half that width of its own, none on `earth-1` at 1,440 px |
| how far a drawing's first tile looks for its nearest neighbour | `graffiti/picture.ts` `NEIGHBOUR_REACH` | 0.05 rad | `length(0.05)` |
| the step of a tile with no neighbour within it | `graffiti/picture.ts` `LONE_STEP` | 0.0075 rad | `length(0.0075)` |

**Left as they are**, since they are not the ground's: the heatmap's readout, a degree of latitude and longitude (a
place's name; on a board as coarse as `fixture-1` a degree can hold no tile and reads "sea", but the Home reads only the
world's board); the heatmap's glow (2 pixels, three times: a softening of the picture); a drawing's margin (4 tiles,
1 in the list, measured in the drawing's own tiles); the Game rules' counts, which the server reads.

## Tests on a second board

`src/testdata/boards/fixture-1/` is the tests' second board, written by the game page's `npm run map:fixture`: a
coarse earth at `cols` 31, 2,998 tiles, 9.7 times as far apart as `earth-1`'s. **It is never bundled**: `npm run map`
does not see it, and only `testBoards.ts` reads it, from the file system (`EARTH_1`, `FIXTURE_1`, `runsOf`,
`placesOn`). The tests rebuild it to its hash, scale its numbers, draw its heatmap and its drawings, and play a
Home and a replay across a switch from `earth-1` to it.
