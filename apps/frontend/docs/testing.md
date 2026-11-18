# Testing

`vitest`, co-located as `*.test.ts(x)`. The suite runs on **node by default** —
most of it is domain logic with no DOM. Files that render components opt in with
a `// @vitest-environment jsdom` comment on the first line, so the rest are not
slowed down by a jsdom each.

`TileField` is testable without a GPU because `BufferAttribute` is plain arrays,
and `TilePicker` because it picks on the CPU: no test needs a WebGL context.

**Two boards to test on** (`src/testdata/boards.ts`): `EARTH_1`, as the page serves it from `static/`, and `FIXTURE_1`, a
coarse earth cut from it at `cols` 31 (2,998 tiles, about ten times as far apart), which `npm run map:fixture` writes
into `src/testdata/boards/fixture-1/` and the backend's and the backoffice's test data. It is never in `static/` nor in
the bundle: no page plays it. `tilesOfBoard` rebuilds a board's tiles as the page does, `englishLandmassNamesOf` reads
its landmass names. What reads the board (see [Sized by the board](board-scale.md)) is tested on both: today's values
on `earth-1`, scaled on `fixture-1`; and what draws each era on its own board, on a history from one to the other.
**The geometry is tested on every shipped board** besides: `earth-2` and `siege-1` (warped, rotated, in canonical
order) rebuild to their hashes (`board.test.ts`), put every tile in its own cell (`latticeCells.test.ts`) and keep the
chunks whole (`tileChunks.test.ts`); `BOARDS` from `boardsAsset.ts` names them all.
