# Virus

A virus (`BonusReward` kind `virus`) takes one enemy tile, then spreads on its own into the land of every enemy flag next to it: the server's
rules, in the backend's [The virus](../../backend/docs/virus.md). No box draws one yet; the operator's grant, and the
fake's `giveVirus()`, give one. A hand holds as many as the rules say (`BonusRules.viruses`, one in production for
now), `Charges.viruses`, counted on the slot once they stack, and a use spends one.

**It is aimed or put away, like the bomb.** The inventory's virus slot aims it (`Globe.setVirusAimed`), the slot or
Escape puts it away, and aiming it puts the bomb away and turns every switch off (and the reverse). While aimed, a tap on
a tile uses it instead of clicking: `Infector.useVirus(tile, flag)`, `ClickService/UseVirus`. The globe refuses a tile
of no other flag, or with a shield, before the call ("A virus takes only an enemy tile with no shield", the notice
`virusNoEnemy` over the slot), and the reach refuses a tile out of it as for a click. The virus stays aimed after a
refusal. The backend takes one off the hand at once and gives that one back unless the server says it holds none: a
`VirusRefusal` (a shield placed meanwhile) is `VirusRefusedError`, said with the same notice.

**Its hint says what it does**, on a hover and in the bubble of its first uses: it takes the tile, then spreads into
that flag's land on its own, up to `BonusRules.virusInfections` tiles (`GetChargeRules.virus_infections`, read again
on each `rulesChanged`). The owner, 2026-11-16: "Tap an enemy tile with no shield" explained nothing, and a phone shows
only that bubble. The shield is said by the refusal.

**Its slot is a kind like the others** (see [Charges](charges.md#the-inventory)): on the bar while one is held, in All's
tray always, after the shields among kinds of one level. The bar holds the kinds held up to its room, so a sixth kind
needs no smaller slot.

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
is infected, the loop keeps drawing (`pulsing`), until nobody has touched the page for a minute: then the pods stop
where they are (see [Drawing only when something changed](render-on-demand.md)).

**No ghost while a virus works**: while one is aimed or any tile is infected, the ghosts stand down and their tiles go
back (see [Ghosts](ghosts.md)).

**In the fake**, a virus spreads every 4 s, over the tiles within a spread's ring, up to 32 infections.
