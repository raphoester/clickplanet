# Mine

A mine (`BonusReward` kind `mine`, `mines` a box gave) waits on a tile of the player's flag, and only that flag's
players see it. When another flag takes the tile, it goes off: the tile and the rings around it go to the mine's flag,
a shielded tile loses one shield and keeps its flag, and every mine the blast reaches goes off too. The server's rules are in the backend's [The mine](../../backend/docs/mine.md).
No box draws one yet; the operator's grant, and the fake's `giveMine()`, give some.

**It is aimed or put away, like the virus.** The inventory's Mine slot (`2/3`, against `GetChargeRules.mines`) aims
it (`Globe.setMineAimed`), the slot or Escape puts it away, and aiming it puts the bomb and the virus away and turns
every switch off (and the reverse). While aimed, a tap on a tile of the flag lays a mine (`Miner.layMine`,
`ClickService/LayMine`) instead of clicking, and **it stays aimed while a mine is held**, so mines are laid together.
**A tap on a tile of another flag, or of none, is a plain click**, as with shield on (`layingOf` in `domain/mines.ts`):
it takes the tile, "Tile taken. Tap it again to lay a mine" (`mineTaken`) over the slot, and the next tap lays the
mine; a click a shield stopped says `mineNotYours`. The globe refuses a tile of the flag that holds a mine or a trap
already, or a capital while the knock-out plays, before the call ("A mine goes only on your flag's tile with no mine or
trap", the notice `mineNotYours`): a tile holds one mine or one trap (see [Trap](trap.md)). The backend takes one mine off the hand at once and gives it back unless the server says it holds
none: a `MineRefusal` is `MineRefusedError`, said with the same notice.

**The flag's mines are read, never public.** `GetMap` carries none. The planet backend reads `GetMines` as each stream
opens and each time the flag it plays for changes, and hands the answer to the globe (`BonusHandlers.onMines`); the
globe keeps it only when it is the flag the page plays for, and forgets every mine when the page changes flag. A mine a
teammate lays comes as `mine_laid` (`onMineLaid`), on the page's stream alone.

**A mine is a tile value under the one version rule** (`FlagMarks` in `MapTiles`, the traps' too, see [The counts are the
server's](counts.md)): each mine is kept with the map's version it is known at. A lay counts no version, so a mine is
kept only while its tile has not changed since; any change of its tile's owner after it takes it off (a tile update, a
blast, an enclosure, a virus, a knock-out, a read that finds another owner), and so does a bomb's strike. A read
replaces the mines known before it, but one laid after it. A map of another epoch forgets them all.

**One `mines_exploded` event per explosion, and no tile update for its tiles** (`minesExplodedOf`, heard as
`onMinesExploded`, after the updates that came before it, so the enemy's take lands first). `MapTiles.applyExplosion`
gives each blast's taken tiles to its flag and sets each struck tile's shields, a tile as the last blast that reached it
left it, and takes every mine off. The struck shields play as a click's strike does. Each blast is drawn
as a bomb's at its impact (`blasts.start` started one fall ago), at `mineBlastRadius(GetChargeRules.mine_rings)`, with
the bomb's sound, full for a mine of the page's flag. So the enemy sees why its click came back.

**A mined tile is drawn with its mine** (the pulse slot's sixth bit, `MINE_MARK`, 32): a dark dome over the flag, its
rim in `--mine` olive, four studs and a red light. It is still: a mine can wait days, and a beat would keep the loop
drawing all that time (see [Drawing only when something changed](render-on-demand.md)).

**On a shielded tile the mine steps down to the shield's tip**, at half its size, as the count fades in (16 to 22px,
see [Shields](shields.md)). In the middle it covered the count. Its light grows there, as at that size the red dot is
what says "mine". Under 16px the shield shows no count, so the mine stays in the middle. A capital's tile stacks the
crown, the count and the mine, top to bottom.

**No ghost on a mine** (see [Ghosts](ghosts.md)).

**In the fake**, every take by another flag sets a mine off a moment after, its blast 2 rings across the tiles within a
spread's reach, and `tripMine()` has an enemy take one of the page's flag's mines.
