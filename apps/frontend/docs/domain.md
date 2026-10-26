# `src/domain/` — the logic worth testing

- `holdings.ts` — `Holdings`, what each flag holds of the map and of each
  landmass, as the server counts it: every count on screen comes from it. Each
  row takes only a newer version. See [The counts are the server's](counts.md).
- `mapTiles.ts` — `MapTiles`, the flag and the shields of every tile: a value
  from a batch, the stream or a bomb is taken only when its version is newer than
  the tile's, and a fortify waits for the read of a tile older than the stream.
  `tileRanges` cuts the map into the batches. `mapClock.ts` — `MapClock`, the
  server start the versions are counted in, and the stream's opening.
- `tileOwnership.ts` — the flag on each tile, and **your own clicks, painted
  before the server has agreed to them** — see [Rolling back a refused
  click](click-rollback.md). A read is what a rollback goes back to; the stream
  settles a click. It counts nothing.
- `leaderboard.ts` — `rankCountries`, a pure sort over the server's counts. Ties break
  on country code so equally-placed rows stop swapping.
- `tileDeltas.ts` — what the leaderboard floats beside a tile count as "+3" or
  "-2". `takeInChanges` folds one board into the badges already up, so a country
  on a run keeps one badge counting up instead of flashing "+1" three times, and
  a country that wins a tile back and loses it again drops its badge rather than
  reading "+0". `expireBadges` retires one `DELTA_HOLD_MS` after its last tile.
  **Both hand back the map they were given when nothing moved**, so a quiet
  leaderboard costs no render.
- `countries.ts` — the country list, keyed by code.
- `chatLog.ts` — `addMessages`, the merge of the history fetch and the live
  stream into one bounded list. **The two sources overlap**: your own message
  arrives twice (the `SendMessage` response and the broadcast that follows), and
  history is fetched after the stream is already open, so it is deduplicated on
  message id, ordered on the time the server stamped, and capped at 200. It
  returns the array it was given when nothing was added, so an echo of something
  already shown costs no render. `unreadSince` counts what arrived after a given
  id and `idsSince` names those messages, for the sound and for highlighting
  them once they are on screen. `unseenAfter` counts the messages and
  announcements after a time, for the badge, and `newestAt` is the time of the
  newest line. `nameSentUnder` is
  the name the server gave the latest message this client sent — the only way
  it learns a guest's name, and only ever shown. `accountSentUnder` is that
  message's account: how a guest's client learns which account it is.
  `startsGroup` says where a new author's run begins: another account, or the
  same account under a new name. `quoteOf` is a message as a quote, for the
  composer and the fake.
- `authorColor.ts` — `NAME_COLORS`, the 12 a player with a username may pick
  from (`player.v1.NameColor`, one hue each), and `hueOf`, the hue of a pick.
  No pick, or a color this build does not know, has no hue. **Only the hue is
  chosen**: the saturation and the lightness are fixed in the CSS, so no pick
  can produce a colour that is unreadable against the dark panel.
  `app/chat/authorStyle.ts` turns a line into the style: a guest, or a name with
  no hue, gets `--author-chroma: 0`, which every rule multiplies its saturation
  by. So **guests are grey** whatever they hold, and **so is a player who has
  not picked a color**: there is no color from the name, so the grey is a
  reason to pick one.
- `streak.ts` — `streakShown`: a flame is drawn from a streak of 3 days. Every
  player of today has 1, so a short run would mean nothing.
- `shareCard.ts` — everything about a shared image that is decided before a
  pixel is drawn: the `?f=<code>` link, the text that rides with it, the line
  under the flag, and the size the card comes out at. See [Sharing the
  globe](sharing.md).
- `shields.ts` — `TileShields`, how many shields stand on each tile, and
  `outcomeOf` and `placementOf`, what this player's click does to a tile. See
  [Shields](shields.md).
- `clickOrDrag.ts` — `ClickOrDrag`, whether a press was a click or a drag of
  the globe. The browser sends `click` after a drag too, so turning the globe
  claimed the tile under the cursor on release. A press that moves more than
  6px (12px for a finger, which rolls as it lifts), or that a second finger
  joins, is a drag, and `globe.ts` drops the click that ends it — the bonus box
  included. **A click lands where its press lifted** (`pointerup`), not where
  the `click` event says: iOS puts the click of a quick second tap where the
  first tap was (WebKit bug 216681, set off by the `dblclick` listener React
  keeps on its root), so a tap on each of two tiles side by side took the first
  one twice.
- `tilePeek.ts` — `TilePeek`, when a pointer asks about a spot, a tile or in orbit a
  landmass: a mouse resting on one, or a finger holding one still. `cardPlacement.ts`
  puts the card beside the pointer. See [Who holds a tile](tile-holder.md).
- `flight.ts` — `territoryShotOf`, the shot a click in orbit flies to, and `flightAt`,
  the camera on the way. See [A territory, from orbit](territory-card.md).
- `territory.ts` — what the territory card says: `rankHoldings`, `freeTilesOf`,
  `sharePercent`, and `sameTerritory`, which spares a render when nothing moved. See
  [A territory, from orbit](territory-card.md).
- `lattice.ts` — the points of the detail-300 icosahedron, in three's order and to the bit, and
  `tilePositions`, the positions of the ones a landmask keeps. `landmask.ts` — the landmask file,
  `landmaskOf` to make one, and `rebuildTilePositions`, which checks the hash. See [Tile
  positions](tile-positions.md).
- `warnOnce.ts` — for things that would otherwise warn on every frame.
