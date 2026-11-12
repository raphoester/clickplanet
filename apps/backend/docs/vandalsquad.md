# The vandal squad (`internal/vandalsquad/`)

**It flags the swastikas players paint on the map, and acts on none.** Every minute it reads in planet's ledger what
each player painted in the last hour, splits the tiles each still holds into drawings, and keeps each one that reads
as a swastika, with a Warn line in the log. Reverting and banning stay the operator's: `planet_revert_player`,
`antibot_ban_player`.

```
planet ◀─ InternalService/GetLedger(the last hour) → the takes ──────────── vandalsquad: what each player painted
planet ◀─ InternalService/GetTiles(ids) → flag, holder, point, neighbours ── vandalsquad: what is on the map now
                                     └─▶ a swastika that shows → Warn line + the graffiti table → ListGraffiti
player ── PlayerForgotten ──────────────────────────────────────────────▶ vandalsquad: the painter is blanked
```

```
internal/vandalsquad/internal/
  paint/                        Take, Painting: each player's latest take of each tile, by flag (PaintingsOf), and
                                the tiles of it the player still holds (Held)
  drawings/                     TileID, Point, Tile, MapTile, Drawing (Links, Flats, Rim), Split, Visible
  swastikas/                    ReadingOf: whether a drawing reads as a swastika; testdata/drawings.json, the real ones
  graffiti/                     Graffiti (Snapshot, Anonymized), Sighting, Patrol (Since, Note), Storage, LinkOf
    usecases/patrol_usecase/    UseCase: one patrol; Runner, a patrol every interval
    usecases/anonymize_graffiti_usecase/   a forgotten account's graffiti, kept as nobody's
    rpc_planet/                 Planet: Takes from GetLedger, page after page; Tiles from GetTiles, 10,000 a call
    postgres_graffiti_store/    the graffiti table; inmemory_graffiti_storage/, the tests' (testing tag)
    log_graffiti/               Reporter: the "graffiti flagged" line, at Warn
  subscribers/
    player_forgotten_subscriber/   player.v1.PlayerForgotten → anonymize (group vandalsquad-graffiti-forgotten)
    log_subscriber/             logs an event it refused
  migrations/                   the vandalsquad schema
  vandalsquadv1controller/      AdminService (bag)
    list_graffiti_handler/graffiti_query/   PostgresQuery: the 500 newest graffiti
```

## A patrol

- **Every minute** (`interval`, the first at boot), the patrol reads planet's ledger over the last hour (`window`,
  `planet.v1.InternalService/GetLedger`), about 8,000 events, one page (190,000 a day measured 2026-11-10). A change
  is a take when an account made it and it gave the tile a new flag: a shield struck or raised and a bomb's clear are
  not. **Each player's latest take of a tile counts**, so a tile it took again for another flag is painted with that
  one. Nothing is kept between patrols, so a restart loses nothing: the next patrol reads the same hour.
- For every tile of every painting of 12 tiles or more (one player, one flag), it asks planet the tile's flag and the
  account whose take holds it now (`planet.v1.InternalService/GetTiles`). **A tile taken since, cleared by a bomb or
  reverted is no longer the player's drawing**: only the tiles it still holds for that flag are read.
- They are **split into drawings, the tiles that touch each other** (`drawings.Split`), and each is read
  (`swastikas.ReadingOf`). For each one that reads as a swastika, the patrol asks planet the flags of the tiles around
  it, and keeps it only when it **shows** (`drawings.Visible`): when at most half of the tiles touching it wear its
  flag. Players painting over a swastika on their own ground leave fresh tiles of its very shape; around them is
  their own flag, around the original another's. On the drawings of 2026-11-06 the originals had none of their flag
  around them and the paint-overs 57 to 100%.
- **A graffiti seen again is one graffiti** (`Patrol.Note`): the same player, flag and kind on some of the same tiles
  keeps its first flagging and takes its latest tiles, link and sighting. The patrol reads the painter's graffiti,
  notes the sighting, and saves it. It is told once, after it is saved (the Warn line `graffiti flagged`, with the
  account, the flag, the tiles and the link), and listed every time. A save that fails ends the patrol; the next one
  sees the drawing again and tells it then.

## What it keeps

**The graffiti, in postgres, for good** (`vandalsquad.graffiti`): its id, its painter's account, its flag, its kind
(the enum `graffiti_kind`), its tiles, when it was first flagged and last seen, and its link. A row is upserted whole
(`Storage.Save`); the store finds a painter's graffiti (`PaintedBy`) and nothing else. `ListGraffiti` reads the 500
newest, newest first, with no domain object in between (see [Reads are queries](queries.md)).

**A forgotten player's graffiti stays, as nobody's.** On `player.v1.PlayerForgotten` the account is blanked
(`Graffiti.Anonymized`, the column NULL): the drawing, its flag, its tiles and its link stay, as the support notes
keep their text (see [Support](support.md)). A graffiti with no painter is never seen again: the same drawing, still
on the map, is flagged anew under the account that holds it.

## What reads as a swastika

**Every rule was read off real swastikas painted on the map**, never off a drawn template: a rule tuned on synthetic
right-angled swastikas caught none of the real ones. Players draw them small (16 to 34 tiles), in strokes one tile
wide, along the grid's lines or zigzagging across them, with arms and hooks of uneven length, each hook turned 60, 90 or
120 degrees on screen. So the reading follows the tiles and measures no right angle and no symmetry:

- **12 to 150 tiles, in one piece.**
- **Strokes one tile wide**: at most 5% of the tiles have five or six neighbours in the drawing. A blob of conquest
  has them all through its middle.
- **A centre with four ends.** From a tile with three neighbours or more, the walk out (breadth first) finds the ends
  of the strokes: a group of touching tiles as far from the centre as each other, with nothing farther next to them.
  On a hex grid a one-tile hook at 60 degrees and the corner where a zigzag meets a straight stroke both make such a
  pair; the corner's stroke goes on past it, the hook's does not. Of an end's tiles, the one whose path turns the
  most stands for it: the hook, not the arm's last tile.
- **Each branch turns once**: its corner is the tile farthest from the line between the centre and its end, with one
  step of arm and one of hook at least, and **all four turn the same way**, by 40 to 140 degrees (58 to 121 on the real
  ones).
- **The arms leave in four directions**, 45 to 135 degrees apart (46 to 122).
- **The branches are of a size**: the longest at most three times the shortest (twice at most on the real ones; four
  and six times on a path with stubs and on a wheel with spokes that the other rules let through).
- **The cross shows**: at most two of the arms are a single tile (one at most on the real ones; three on a hook-shaped
  blob the operator judged none).
- **The branches are the drawing**: at least 80% of its tiles lie on the four paths (88 to 97% on the real ones, 51 to
  65% on the scribbles the other rules let through), and every tile is on them or next to them (85%).

The reading looks at the drawing as the globe shows it, around its middle, with the north up (`Drawing.Flats`). It
uses angles and steps only, never a length on screen, so a tile's spacing, which varies by a quarter over the globe,
changes nothing.

**Measured, on 2026-11-07**: the 32 hours of replays the clips job saved, and season 0's history (its map every 30
minutes, and every 10 seconds over the Final Battle), every connected patch of one flag of 12 to 150 tiles
in them, 25,962. The reading flags 24, the 24 drawings an eye review of every candidate found to be swastikas, and nothing else.
The four drawings the operator was asked about it reads as none, as the operator judged them. `swastikas/testdata/drawings.json` holds
those swastikas (and two of them drawn again along the grid's other lines, elsewhere on the globe) and 79 drawings that
are none, each tile as `GetTiles` answers it with only its neighbours in the drawing. **A drawing judged later goes
there, with its reason**, and the tests say which rule it breaks.

## What it does not do yet

- **No revert, no ban, no push.** The list and the log are all. Acting alone (planet's revert, the antibot's ban, a
  push to the operator) is a policy change of [Chat moderation](../../../docs/moderation.md)'s, where every ban and
  revert is the operator's, and its own step.
- **No text.** "HEIL HITLER" is letters; reading them needs a model, which the squad does not call.
- **No swastika in strokes two tiles wide or more**: none was ever painted, so none is in the samples.
- **Nothing painted during an arena battle**: the ledger's read leaves out what was played on an arena's map.
- **A drawing painted over more than the window, or by several players, is seen in part or not at all.**

## Configuration

`vandalSquad` (see [Configuration](configuration.md)): `enabled`, `database` (schema `vandalsquad`, one connection in
production), `redis` (planet's, for `PlayerForgotten`), `gameUrl` (the page a link opens,
`https://clickplanet.lol/play`), `window` (1h, at most the 72h the ledger reads at once) and `interval` (1m, at most
the window). It asks planet over the internal listener. Off, it is never built, and `vandalsquad.v1.AdminService`
404s.

`vandalsquad.v1.AdminService/ListGraffiti` is on the admin listener, so the connector offers it as
`vandalsquad_list_graffiti` (see [Claude as a connector](connector.md)).
