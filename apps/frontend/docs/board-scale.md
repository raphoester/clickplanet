# Sized by the board

**Every count of tiles, length and zoom the game was tuned on is `earth-1`'s, and reads the board it is drawn on**
(docs/boards.md, "Rules at the new density"). A board's tiles are its own distance apart: `earth-2`'s half as far,
`siege-1`'s about three times as far. What pays keeps its tile count (a bonus never pays more than a full click bank),
and what describes the map scales.

## The rule

`domain/boardScale.ts`, the same rule as the backend's and the backoffice's:

- `EARTH_1_SPACING = 0.004063093462255102`, `earth-1`'s manifest spacing, the reference.
- `scaleOf(board)`: `length` = the board's spacing over `earth-1`'s, `density` = `(EARTH_1_SPACING / spacing)²`.
- `scaledCount(n, scale)`: a count of tiles on `earth-1`, as many on the same ground: `max(1, round(n × density))`.
- `scaledLength(x, scale)`: a length on the globe of radius 1: `x × length`.
- `scaledZoom(z, scale)`: a camera zoom where a tile is as wide on screen as on `earth-1`: `z / length`.

On `earth-1` the ratio is exactly 1, so every value is today's to the bit (`boardScale.test.ts`).

**Functions of the board, not module state.** The game page plays one board for its life (a change reloads it: see
[The board the page draws](board-reload.md)), but the archive, the clips and the player's video draw several boards in
one page. So each threshold is a function of a `BoardScale`, and each globe takes its own from the board it is built on.

## What reads it

| what | where | `earth-1` | the rule |
| --- | --- | --- | --- |
| fortify news | `domain/fortify.ts` `newsFromTiles`, the fake chat's `announceFortify` | 50 tiles | count |
| clip: a landmass too small to flip | `domain/clip/look.ts` `minLandmassTiles` | 20 | count |
| clip: a flip read from far | `look.ts` `readableFlipTiles` (`Flip.readable`) | 100 | count |
| clip: a country fought over | `clip/changes.ts` `foughtOverLeast` | 400 | count |
| clip: a big country, a place | `clip/findings.ts` `bigCountryTiles` | 1,000 | count |
| the least landmass a flag is painted in | `viewer/borderField.ts` `leastPaintedTiles` | 4 | count |
| the heatmap's cell | `archive/heatmap.ts` `cellDegreesOf`, its widest cell | 0.4°, 64 px | length |
| max zoom | `viewer/zoom.ts` `maxZoomOf` | 50 | zoom |
| a tile's disc and spacing on screen | `viewer/pointSize.ts` `tilePointSize`, `tileSpacing`, and all that reads them: the handover to the painted flags, orbit, a territory flown to, the clips' and the video's dive | 1.5, 1.98 px at zoom 1 on 1,000 px | length |
| the neighbours' reach | `domain/neighbours.ts` `Neighbours` | 0.0046 | length |
| a mine's blast | `domain/blast.ts` `mineBlastRadius` | 0.004 a ring | length |
| a spread onto nothing | `viewer/clickEffects.ts` | 0.004 | length |
| an infected tile drawn nearer | `viewer/pointSize.ts` `virusLiftOf`, the display vertex shader's `virusLift` | 0.012 | length |
| a covered flag's least half-height | `pointSize.ts` `leastFlagRadiusOf`, the territory fragment shader's `leastFlagRadius` | 0.004 | length |
| the fake's bomb, sea, spread ring and rally | `backends/fakeBackend.ts` `lengthsOf` | 0.032, 0.004, 0.004, 0.072 | length |
| the fake's share of the map | `backends/fakeStandingsBackend.ts` | its board's tiles | |

The bomb's radius and the rally's are the server's, read with the rules, so they follow the board on their own; so do
the clips' solidity (measured from the tiles' own spacing), and the cells of the territory layer and the pick.

## What does not

- **What pays, or counts clicks**: a fortify target's 25 tiles missing (`fortifyTarget.ts`), the front guide's 50 takes,
  the ghosts' raids of 3 to 8 takes, the antibot's thresholds.
- **What is geographic**: the clips' 10° cells, their fronts' radians and their framing (`camera.ts`'s closest framing,
  `tour.ts`'s stops), the bridges' dots (about 38 km apart), the chunks the page culls by (0.1 rad).
- **What measures work**: the map's batches of 10,000 tiles, the ghosts' search of 4,000 tiles.
