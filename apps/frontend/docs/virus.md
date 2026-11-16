# Virus

A virus (`BonusReward` kind `virus`) takes one enemy tile, then spreads into that flag's land on its own: the server's
rules, in the backend's [The virus](../../backend/docs/virus.md). No box draws one yet; the operator's grant, and the
fake's `giveVirus()`, give one.

**It is aimed or put away, like the bomb.** The inventory's virus slot aims it (`Globe.setVirusAimed`), the slot or
Escape puts it away, and aiming it puts the bomb away and turns every switch off (and the reverse). While aimed, a tap on
a tile uses it instead of clicking: `Infector.useVirus(tile, flag)`, `ClickService/UseVirus`. The globe refuses a tile
of no other flag, or with a shield, before the call ("A virus takes only an enemy tile with no shield", the notice
`virusNoEnemy` over the slot), and the reach refuses a tile out of it as for a click. The virus stays aimed after a
refusal. The backend takes it off the hand at once and gives it back unless the server says it holds none: a
`VirusRefusal` (a shield placed meanwhile) is `VirusRefusedError`, said with the same notice.

**The slot shows only while a virus is held or aimed.** The bar keeps its five slots for everyone else, so it needed no
phone redesign. **A sixth slot shrinks the slots, never widens the dock** (`Inventory.css`, `:has`): the dock's width
sets the chat's room (`--dock-width`), and the click count never shrinks, so a row wider than five slots slid under it.
From 1280px the slots are 60px at least with a 4px gap (439px against 421px in French, the widest names); below, 40px
with a 4px gap; on a phone, 38px with a 3px gap (243px against 240px), and the meter keeps its bar at 360px wide.

**One `virus_spread` event per use or tick, and no tile update for its tiles** (`virusSpreadOf`, heard as
`BonusHandlers.onVirusSpread`, after the updates that came before it). `MapTiles.applyInfection` gives the tiles it took
to its flag at its version and infects them, sets the shields it struck, and, once it `ended`, cures every tile it
infects. **A tile's virus is a tile value under the one version rule** (`TileInfections`, see [The counts are the
server's](counts.md)): a map read sets it from `GetMap`'s `infected`, and any change of owner cures it, as on the server
(a tile update whose flag changed, a blast, an enclosure, a knock-out). `infectionMarksOf` turns a list of changes into
marks with each tile's last change winning, since a virus that ends in the event that took a tile cures it at once.

**An infected tile is drawn as the bonus's icon** (the pulse slot's fifth bit, `INFECTED_MARK`, 16): its flag, the same
size as every tile, in a `--virus` membrane, with 8 stalks and pods past it that turn, each tile from its own angle
(`vSeed`, from its position), the pods beating one after the other. The sprite is 1.8 times the tile, to hold the pods,
and is drawn 0.012 nearer the camera, so its pods lie over the tiles next to it whatever their depth or draw order; the
pods' edges are hard, since a soft edge would write depth where it hardly draws. Under 9 CSS pixels a tile has no room
for pods: it is washed in `--virus` alone. The owner, 2026-11-16: the first look, a tile washed in purple that swelled,
was "pretty lame", and a flag shrunk into a smaller body looked smaller than the other tiles and weird. While any tile
is infected, the loop keeps drawing (`pulsing`).

**No ghost while a virus works**: while one is aimed or any tile is infected, the ghosts stand down and their tiles go
back (see [Ghosts](ghosts.md)).

**In the fake**, a virus spreads every 4 s, over the tiles within a spread's ring, up to 32 infections.
