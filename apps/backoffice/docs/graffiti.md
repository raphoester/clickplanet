# Graffiti

`src/graffiti/`, under Moderation. **Each drawing the vandal squad read as a swastika, drawn, for a person to judge
before acting on its painter**: the squad flags by rules read off real swastikas (see the backend's
[The vandal squad](../../backend/docs/vandalsquad.md)), and a rule is no eye. `vandalsquad.v1.AdminService/ListGraffiti`
answers the 500 newest, newest first, and the page asks `player.v1.AdminService/GetPlayers` for their painters' names
and `antibot.v1.AdminService/InspectPlayer` for where each painter stands.

- **An inbox, as the reports are** (see [Acting on a report](reports.md)): the list on the left, the one open on the
  right from 1280px, one at a time below. A row is the drawing small (its tiles alone, a tile of margin), the flag it
  was painted with, its painter's name, "×2" when the painter has more than one, how long ago it was flagged, its
  size, and whether the painter is banned now. **All**, or **Painter not banned**: a graffiti whose painter is banned
  is dealt with. The squad keeps no verdict, so a drawing judged none stays in All.
- **A graffiti has no id** in the proto. The squad tells one from another by its painter, its flag and its tiles, and
  keeps its first flagging when it sees it again, so the address bar names it by those
  (`#/graffiti/<flagged at, ms>-<flag>-<account id>`, `keyOf`). One the squad no longer lists says so.
- **The drawing open is drawn large**, as the globe shows it from straight above its middle, north up and east right,
  with four tiles of margin: its tiles in the text color, the land around them in the map's grey, the sea left blank.
  Under it: its painter (the name opens the player page, with every tool; the id with Copy), its flag, its size, when
  it was first flagged and last seen, a link that opens the globe on it, where the painter stands, and the painter's
  other graffiti, one click each.
- **The tools are the painter's Ban and Revert**, the player page's own (`BanAction`, `RevertPanel`): the ban asks
  for how long (the ladder's next step by default) and reads where the painter stands again; the revert runs its dry
  run first. Ban first, or the painter paints again behind the revert. A deleted account's graffiti has no tools.
- **What is drawn is what the squad last saw**, its tiles at `seen_at`, not the map now: a drawing painted over or
  reverted is still drawn, and the squad reads only the last hour of takes, so one older than that is not seen again
  even when it still stands. The globe link shows the map now.
- **The squad off** (its service absent, `unimplemented`) says so instead of failing.

**Each graffiti is drawn on the board it names** (`Graffiti.board`): its tile ids are places on that board alone. The
page reads each board the list names once (`boardsOf`, `placesByBoard`, over `loadPlaces`: see
[Boards on the page](boards.md)), and the detail says the board. A board the page does not carry fails alone: a notice
above the list says which, its drawings say their tiles could not be placed, and the rest are drawn. Every graffiti
names `earth-1` today.

## How a drawing is drawn

`picture.ts`, `pictureOf(places, tileIds, margin)`, on the places of the graffiti's board (`src/boards/places.ts`):

- **The drawing's middle** is the mean of its tiles' points on the sphere; east and north there make the plane it is
  drawn on, each tile its point's projection (`x = p·east`, `y = p·north`), so north is up wherever it is, a pole and
  the antimeridian included. The squad's own link is the same middle.
- **The frame** is the drawing's farthest tile from the middle plus `margin` steps of the lattice, the step read off
  the first tile's nearest neighbour (0.17° to 0.25° on `earth-1`), looked for within 0.05 rad, or 0.0075 rad for a tile
  with none there; both are `earth-1`'s lengths, read at the board's spacing (`scaleOf`), so a coarse board's lone tile
  is framed with the land around it too. The margin is in the drawing's own tiles, on any board.
- **The tiles around it** are found by one-degree bands of latitude and longitude (`bandsOf`, built once a page load),
  not by reading the 260,000 tiles for each drawing.
- **Each tile is its Voronoi cell**: the circumcentres of the tile and each pair of its neighbours in turn, a hexagon.
  The map holds land alone, so a tile on the coast misses the neighbours the sea would be; each missing one stands where
  the lattice would put it, a step away at 60° from the next, so a drawing on the shore is drawn whole.
- **`groundOf` draws the land around a place the same way**, with no drawing: a frame of a given width in km, the
  cells of every tile in it, and where other places fall in it. Flag submissions draws a pin on it (see
  [Flag submissions](flag-submissions.md)).
- `npm run demo` lists four drawings of the squad's test samples (`swastikas/testdata/drawings.json`), painted by
  made-up players: three swastikas, two of them by one player, and one the operator judged none.
