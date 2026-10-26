# The counts are the server's, and every value has a version

**The browser counts no tile.** The leaderboard, each flag's share of the map, the anthem, the share card's rank, the
painted flag of each landmass (its leader and its share), the territory card's flags and the fortify target all come
from the server's counts (`ClickService/GetHoldings`, then `holdings_changed` on the stream, at most once a second).
The tiles a player clicks still come from the map batches (`GetMap`), but nothing is counted from them. See the
backend's [The map's version](../../backend/docs/map-version.md).

## `domain/holdings.ts`: what each flag holds

`Holdings` keeps two kinds of row: the tiles of each flag on the map, and each landmass whole (the tiles each flag
holds of it). A read (`read`) lists every row; a change (`change`) lists the rows that moved, a flag that left the
map at 0 and a landmass nobody holds any more with no flag. **Each row keeps the version it was set at, and takes
only a newer one.** A row at 0 is kept with its version, so an older copy cannot bring it back, and a newer read
drops a row it no longer lists. It answers `counts()` (the board, through `rankCountries`), `holdingsOf(landmass)`
(the territory card) and `leaderOf(landmass)` (the painted flag and the fortify target; ties on the flag's code).

## `domain/mapTiles.ts`: one rule for every value of a tile

`MapTiles` holds `TileOwnership` (the flag) and `TileShields` (the shields) of every tile, and the version of the
last value each tile took (`Float64Array`, -1 for a tile nothing told yet; the server's `uint64` arrives as a
`bigint` and fits a double: a server start never counts 2^53 writes). **A map batch, a tile update, a bomb's clear
and its strikes are taken for a tile only when their version is newer than the tile's.** That one rule is why an
old batch never paints over what the stream said, why a later take wins over a clear held back for the bomb's
impact, and why a value heard twice changes nothing. A bomb says the shields each struck tile kept, so no event
changes a value by one, except the fortify.

**A fortify adds a shield** to each tile of its landmass below the most, with no tile list. It is taken at once on a
tile whose range was read at the stream's opening or after; a tile of a range older than that waits for its next
read (`waiting`), then gets the shield if that read is older than the fortify. The waiting fortifies are dropped
once every range is current.

`claimedLive` is gone: `TileOwnership` keeps only the player's own clicks in flight (see
[Rolling back a refused click](click-rollback.md)). A read is what a rollback goes back to; the stream settles a
click.

## `domain/mapClock.ts`: which versions compare

A version is `{epoch, seq}`. `MapClock` keeps the epoch the client's versions are counted in and the version the
stream last opened at. **A new epoch means the server started again**: `MapTiles` forgets every tile's version (the
tiles stay drawn) and `Holdings.forget` lets the next read replace every row; both are read again. A read or an event
of another epoch is not taken.

## The load, in `globe.ts`

1. Every listener is set before any read. `listenForOpenings` hears the version the stream opened at; a listener
   that comes late hears at once the version the stream already passed, after its updates were handed over, so a
   read from then on holds whatever it missed.
2. `GetHoldings` and `GetFortresses` are read at once. **`createGlobe` resolves on them**: the globe, the painted
   flags, the leaderboard, a flight in orbit and the territory card work from there.
3. The map batches start with them, eight at a time: on Fast 4G that put the counts in at 1.56s and the tiles at
   2.06s, where waiting for the counts first put the tiles at 2.27s. **A tile is clicked only once they are all
   read** (`Globe.tilesRead`;
   `onLoadProgress` says the share). Until then a click zoomed in does nothing, and `Viewer` shows the loading card
   when zoomed in (`onOrbitChange`); in orbit nothing waits.
4. **A read older than the stream's opening may miss what happened in between**, which the stream never sends.
   So, once the opening is known, every range and the counts still older than it are read again
   `READ_AGAIN_MS` (5s, the reads' `max-age`, so the browser does not hand back the same copy) later, four reads at
   most. The server answers a 304 when nothing moved.
5. **A stream that opens again had a gap**: what is older than its opening is read again at once, the fortresses
   too, and again 5s later while a read comes back older. There is no fixed wait any more. A range that was current
   stays current up to the last version the old stream told (`MapTiles.heard`), and the counts up to the last
   `holdings_changed`: so the reopen of a player's first click, when nothing was written in between, reads nothing.

A tile load that fails after its retries is tried again 5s later, so the loading card does not stay for good.

## The fakes

`FakeBackend` plays the server: the real tile count (the borders blob's), a version on every write, `FakeCensus`
counting what each flag holds and sending `holdings_changed` after every flush of its updates, and an opening at
once. `ReplayBackend` numbers each step it plays from its opening, counts with the same `FakeCensus` (it takes the
borders' landmasses: `ReplayBackend.of(response, landmasses)`), and fills the shields a struck tile kept from its
own map for a replay recorded before the server said them. The clip awaits `Globe.tilesRead` before its first frame.
