# Trap

A trap (`BonusReward` kind `trap`, `traps` a box gave) waits on a tile of the player's flag, and only that flag's
players see it. When another flag takes the tile, it springs: the player whose take it was loses clicks from its bank,
down to minus one bank, and the taker keeps the tile. The server's rules are in the backend's
[The trap](../../backend/docs/trap.md). No box draws one yet; the operator's grant, and the fake's `giveTrap()`, give
some.

**It is aimed or put away, like the mine** (`Globe.setTrapAimed`, the slot or Escape). Aiming it puts the bomb, the
virus, the mine and the rally away and turns every switch off, and aiming any of them puts it away. While aimed, a tap
on a tile of the flag lays a trap (`Trapper.layTrap`, `ClickService/LayTrap`), and **it stays aimed while a trap is
held**. A tap on a tile of another flag, or of none, is a plain click (`layingOf` in `domain/mines.ts`, the mine's rule):
it takes the tile, "Tile taken. Tap it again to lay a trap" (`trapTaken`) over the slot, and the next tap lays it. The
globe refuses a tile of the flag that holds a trap before the call ("A trap goes only on your flag's tile with no trap",
`trapNotYours`); a tile with a mine takes one, and a capital too, as the take that knocks its flag out springs it
first. The
backend takes one trap off the hand at once and gives it back unless the server says it holds none: a `TrapRefusal` is
`TrapRefusedError`. The hints say only what to tap.

**The flag's traps are read, never public.** `GetMap` carries none. The planet backend reads `GetTraps` as each stream
opens and each time the flag it plays for changes (`BonusHandlers.onTraps`), and a trap a teammate lays comes as
`trap_laid` (`onTrapLaid`), as the mines do. **A trap is a tile value under the one version rule**, kept beside the mines
in `MapTiles` (`traps`, the same `FlagMarks` as `mines`, in `domain/flagMarks.ts`): any change of its tile's owner after
the version it is known at takes it off, a bomb's strike too, and a map of another epoch forgets it. The page forgets
every mine and trap when it plays for another flag (`forgetMarks`).

**A trapped tile is drawn with its trap** (the pulse slot's seventh bit, `TRAP_MARK`, 64): an open set of steel jaws,
their teeth in, on two hinges, round a `--trap` plate on a white face (no flag shows inside the jaws). It is still, as the mine is (see [Drawing only when something
changed](render-on-demand.md)). **On a shielded tile it steps down to the shield's tip** at half its size, as the mine
does, and its plate grows there; a capital's tile stacks the crown, the count and the trap, top to bottom. **A tile with
a mine and a trap draws them side by side**, the mine on the left, each at 0.7 of its size (`PAIR_SIZE`,
`PAIR_APART`), in the middle or at the shield's tip.

**`trap_sprung` reaches every page** (`onTrapSprung`, told after the take's own event): two toothed steel jaws slam shut
over the tile in about 150 ms with an overshoot, the tile flashes `--trap` and shakes, and what left the taker's bank
pops and rises ("−10"), the rally's "−0.5" bigger. Gone in about a second (`trapSnaps.ts`, DOM over the canvas, so the
globe's loop draws nothing for it). With reduced motion the jaws show shut and do not move, nothing shakes, and the
count only fades. A metal snap plays (`trap`, under the bomb's switch, see [Sound](sound.md)): full for a trap of the
page's flag or the page's own take, softer for the others.

**The taker's bank drops at once.** A click that springs one answers with the drained bank (`ClickResponse.budget`), and
a take with no answer of its own (a virus's tick, a mine's blast) comes with `yours` on the stream, and the backend reads
`GetBudget` again. **The bank may read below zero, down to minus one bank** (`tokensAt` in `backends/clickBudget.ts`):
the meter says "−7.5 left" in `--trap-light`, refuses every click, and counts the wait to the first click it allows
again; the next-click bar stays empty. A drop of more than a click and a half between two readings flashes the meter in
`--trap` once (`click-budget-drained`).

**No ghost on a trap** (see [Ghosts](ghosts.md)).

**In the fake**, `giveTrap()` gives one, `tripTrap()` has an enemy take one of the page's flag's traps (the snap, at the
enemy's cost), and `enemyTrap()` lays another flag's trap on one of its tiles, which the page's next take of it springs:
the bank drops by 10, down to minus 60.
