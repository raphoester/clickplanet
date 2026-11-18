# Sized from the board

**A number the game was tuned on `earth-1` reads the board it is used on** (PR 3 of [Boards](../../../docs/boards.md),
2026-11-18). A count of tiles is as much ground on any board, a length as many tiles across, and a zoom shows a tile as
wide on the screen. What pays keeps its tile count: a bomb's rings, the spread, an enclosure's most, the missions' goals,
the XP, the antibot's thresholds and the toll stay as they are (docs/boards.md, "Rules at the new density").

## The rule

`cpboards` (`internal/shared/cpboards`) reads every board this app carries (`Files()`, `Boards`, `Named`) and gives each
a `Scale` against `earth-1`, whose manifest's spacing is the reference (`Earth1Spacing`, 0.004063093462255102 rad):

- `Length()` is the board's spacing over `earth-1`'s; `Density()` its square's inverse, the tiles on one of `earth-1`'s;
- `Tiles(n)` is `max(1, round(n × density))`: a count of `earth-1`'s tiles on the board;
- `Distance(x)` is `x × length`, and `Zoom(z)` is `z ÷ length`.

On `earth-1` every value comes out as it was, to the bit: the ratio is exactly 1, and so is the zero `Scale`, which is
`earth-1`'s. `ScaleOf("")` is `earth-1`'s too: an event or an answer from before the boards were named. A board this app
does not carry has no scale (`ErrUnknownBoard`). The page and the backoffice apply the same rule, from the same number.

## What reads it

| what | where | on `earth-1` | the board from |
| --- | --- | --- | --- |
| mission floors | missions `draw.go` (`World.atLeast`): a rival 1,000, a ground 3,000 with 1,000 open, a raid 300 | unchanged | `GetTerritoriesResponse.board`, read with the territories |
| fortify news | chat `announcement.go` `fortifyNewsTiles`, 50 | unchanged | `Fortified.board` |
| largest drawing | vandal squad `swastikas/reading.go` `maxTiles`, 150 (the smallest, 12, stays) | unchanged | `GetLedgerResponse.board` |
| a graffiti's link | vandal squad `graffiti/link.go` `closestZoom`, 20 | unchanged | `Graffiti.board` |
| attack ping zoom | alerts `pings/view.go` `closestZoom`, 20 | unchanged | `TilesTaken.board` |
| a bomb's radius, a rally's ring | planet `charges.Bombing`, `rallies.Muster`: rings × the spacing measured | unchanged | the board in play (`clicks.Played`) |

**Planet's own rules read the board in play** (`clicks.Played`, see [Boards and eras](boards-and-eras.md)): a bomb's and
a rally's radius are their rings times its measured spacing, so they follow the board with no scale.

**The vandal squad waits while an arena plays a board of its own**: it reads the world's takes on the world's board,
and `GetTiles` names the board in play, so a patrol that would read the arena's tiles under the world's ids reads none
(`drawings.ErrOtherBoard`) until the world's map is back.

**Left as they are**, each counting what is no ground: the attack's 20 takes (`pings.AttackTiles`, takes made, as the
antibot's thresholds count clicks), the rally's `charges.rally.rings` (a rule set's, which the operator edits at the
switch: docs/boards.md).

## Tests

`cpboards`' `scale_test.go` (every value unchanged on `earth-1`, and on the fixture: 1,000 tiles are 11, 50 are one, a
zoom of 20 is 2.06); the missions' `TestTheFloorsAreAsMuchGroundOnAnyBoard` and `rpc_planet_world`'s scale; the chat's
fortify on the fixture; the vandal squad's largest drawing, link and patrol while another board plays; the alerts'
view on the fixture.
