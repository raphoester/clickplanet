# The level

A signed-in player has a level, from XP the server counts (see the backend's [Levels](../../backend/docs/levels.md)).
`app/level/` draws it. **The page computes no level and no XP**: it draws what `GetLevel`, the card's
`Progress.level` and the stream's `level_reached` answer, all the same `PlayerLevel` (the level kept, the XP now, and
the two thresholds its bar runs between).

**Held for the reveal.** The level shows as soon as this page is deployed and the server answers one, so it ships with
the rest of the reveal (missions paying XP, the level-up bonus, the unlock levels): its PR merges only with them.

## Where it shows

- **The dock**: `LevelLine`, the level's chip and its bar across the whole dock, under the clicks and the bonuses
  (`ClickBudgetMeter`'s `footer`). The dock grows by the line's height, 16px, and only while there is a level. Picked by
  the owner on 2026-11-16 over three placements shot at phone and desktop width: on the You button (a bar 40px wide
  barely moves), under the clicks alone, and along the status bar. A bar as wide as the dock moves visibly as the
  player takes tiles.
- **The public card**: the level's chip before the name (`PlayerCard`, from `Progress.level`). Not beside names in the
  chat, the roster or the board: the streak and the worn title are already there.
- **The Progress tab and the profile**: `LevelProgress`, a large chip, "Level 12", the bar and the XP on it ("1,366 /
  5,558 XP"), under the worn title on the phone and in the profile's hero on a desktop.
- **The level-up moment**: `LevelUp`, in the announcements queue (`{kind: "level"}`, keyed by the level), one
  full-screen moment at a time. It is the title unlock's stage (`TitleUnlocked.css`, `TITLE_REVEAL`, the `title`
  sound) with the level's number in a gold block where the medal lands. Nothing closes it before it has landed. A jump
  of several levels is one `level_reached`, so one moment. A level reached while no tab is open is never announced.

## The numbers

- **`useLevel`** (`levelStore.ts`) holds the page's own level. **`useLevelFeed`**, which `PlayScreen` calls, reads
  `GetLevel` (`AccountStore.level`, with the click token) when a signed-in account loads and again once its clicks
  settle (`useReadsAfterClicks`, as the titles), and forgets the level when the account signs out or changes. The
  stream's `level_reached` goes to the same store, from `PlayScreen` (`useProgressionEvents`, the titles' stream).
- **`newerLevel`** (`domain/level.ts`): a read sent before a raise never takes the level shown back down.
- **`barOf`** (`domain/level.ts`) is the bar: the XP gained on the level and what the level asks. The server sends the
  XP as it counts it, which passes the next threshold until the raise lands (the bar shows full) or sits under the
  level's own after a stricter scale (empty): the bar clamps.
- **A guest draws nothing**: no read, an empty store, and its card has no level. A signed-in account the server keeps
  no level for draws nothing either, until its first `level_reached`, which shows the level and plays the moment.

## Not here

The XP a mission pays (`GetLevel`'s `mission_rates`, which `ProgressionBackend.level` does not map), the locks on the
bonus slots and the next unlock belong to the missions' and the bonus bar's work. The bonus bar reads `GetLevel` itself
for its locks (`LadderReader.ladder`, `useGatesFeed`, a guest's too, from the session held): two reads of one call after
a run of clicks, which one read could serve.

## The fake

Every fake player has a level from its seed, on the server's curve (`fakeLevels.ts`, the fake's own copy). The fake's ladder
answers the same level, so the bar's locks and the dock agree. `giveXp(500)`
adds XP to the signed-in player and plays the level-up when it passes a level; XP below a level shows at the next
read, after a run of clicks (see [Commands](commands.md)).
