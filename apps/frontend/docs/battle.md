# Battles

**The game shows when the schedule plays a battle.** The backend's metagame keeps a
schedule of rule-set entries, and each entry has a kind (the backend's
[metagame](../../backend/docs/metagame.md)). The client never hears a rule set's
name, only the kind of the entry that plays, from `metagame.v1.MetagameService/GetTurn`:
`battle` or `plain`, since when, and when the running entry ends. A battle is the
one kind with a look.

## Reading what plays

- `backends/turn.ts` is the contract: `Turn` (`kind`, `since`, `endsAt`,
  `description`) and `TurnBackend`, with `turn()` and `changes()`, which tells when
  what it answers may have changed. `turnBackend.ts` reads `GetTurn` as a cached GET
  and tells of a change on a page wake (`pageWakeups`: a phone that locked may have
  slept through a start). A server without the metagame (`Unimplemented`,
  `NotFound`) reads as a plain turn. A turn with no kind, or a kind this build does
  not know, is refused: the switch names every value. `fakeTurnBackend.ts` is the
  fake, with `startBattle(minutes, description)` and `endBattle()`, which tell of
  the change at once.
- **`description` is the entry's, the operator's words for the players**, written
  on the backoffice's Rule schedule page (at most 300 characters, the backend's
  rule). Empty when the operator wrote none, or when no entry runs.
- `app/battle/useTurn.ts` holds the turn: read at mount, read again on a change,
  at the running entry's end, and once a minute otherwise (`domain/battle.ts`,
  `nextReadAt`): an entry at a higher priority may start before the end, and the
  answer is 5 s fresh. A read that fails is logged and tried again at the next
  tick.
- `domain/battle.ts` — `battleOf(turn)`, the `Battle` a battle turn is (nothing for
  a plain turn or a battle with no end), its key (`since:endsAt`), its clock
  (`battleLeft`, `timeLeftTo` from the season clock, stopped at zero), and
  `battleEndsLine`, "Ends at 23:00" in the player's time, with the weekday when the
  end is not today.

## The look

**The battle is a mood** (see [`DESIGN.md`](../DESIGN.md#moods)): `useMood` writes
`data-mood="battle"` on the root while a battle plays, and `tokens.css` redefines
the panels to charcoal and the secondary button to steel under it. The warning tape
(`--tape`) runs along the top of the dock (`ClickBudgetMeter.css`, under the mood)
and of the battle chip. **The globe's halo turns to ember**: `Viewer` hands
`useGlobe` a `halo` (`calm` or `war`), the globe's `setHalo` repaints the
atmosphere's colour (`app/viewer/atmosphere.ts`) and asks for a frame. The sky, the
gold and the text do not move: a coin, a header or a reward looks the same in a
battle.

## The chip

**While a battle plays, the status zone's chip is the battle's** (`BattleChip`),
in the season chip's place and shape (its `season-chip` classes, with its own
colour, glow and tape in `Battle.css`): crossed swords, "Battle ends in", and the
clock to the second. The season chip, or the champion's, is back when the battle
ends.

- On a desktop a hover opens its facts under it, and a press holds them open
  (Escape closes a pressed one; a hover-only one closes when the pointer leaves).
  On a phone it is the right end of the status bar, "Battle" over the two largest
  units, and a press opens the "Battle" sheet (`BattleDetails`): the clock and the
  same facts.
- **The facts are the description, when the operator wrote one, and when the battle
  ends.** The season's countdowns follow, so a player loses no clock while the chip
  is the battle's; not during the finale, where the battle's clock is the season's.
  The facts explain no rule on their own: what a battle changes is the operator's
  description, read from the server.
- The chip's name holds the clock in full, and the visible clock is hidden from a
  screen reader, as the season chip's is.

## The start moment, and the end

**The start plays once per battle per browser** (`BattleReveal`, `useBattleReveal`):
the title unlock's slam, "Battle!", a disc of crossed swords in the battle's
colour, the description and when it ends, then "To the globe" once it settled
(`TITLE_REVEAL.ready`). Close keeps the battle's key in `clickplanet-battle-seen`,
so a reload does not play it again; a player who arrives mid-battle sees it, since
the battle is on. It plays through the announcements queue (below).

**The end is quiet for now**: the mood goes, the halo turns blue again, and one line
in the moments zone says "The battle is over" (`BattleOver`, `useBattleOver`, 5 s),
only when the battle ended while the page was open. A debrief of the battle, with
its numbers from the backend, is planned; it will be an announcement like the
others.

## The announcements queue

**One full-screen moment at a time, in the order they arrive.** `domain/announcements.ts`
and `app/announcements/useAnnouncements.ts` keep the queue: a title unlocked, a
battle's start, a round closed (the cutoff's reveal). `Viewer` announces each as it
learns of it, shows the head of the queue once the map is loaded, and the next
starts when the player closes the current one. An announcement is queued once, by
its key. So a round that closes while the player reads a battle's screens waits
its turn, and nothing is drawn over anything.

The round's "once per browser" and the battle's keep their own rule (`useRoundReveal`,
`useBattleReveal`): the queue only orders what is due.

## In fake mode

`startBattle(minutes = 120, description)` and `endBattle()` on the console (see
[Commands](commands.md)): the mood, the halo, the chip and the moment follow at once.
