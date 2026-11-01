# Titles

A linked account earns titles; the server decides which and keeps them (see
the backend's [Player](../../backend/docs/player.md)). `app/titles/` draws them and `app/account/ProgressTab.tsx`
is the player's own view.

- **Most titles are ranks on a track** (`TitleRank`: the track, the rank's
  number, how many ranks). Conquest counts tiles taken, Devotion the streak,
  Chatter messages, Defense fortified tiles, Bombs bombs dropped, Loot boxes
  caught, Quiz right answers and Missions the missions done; OG stands alone, and so do the season titles
  (Season 0 Gold, Silver and Bronze, a trophy with the place number in their metal, Gold crowned, Season 0 Top 1, 2
  and 3, a star and the place, Season 0 Battle Top 1, 2 and 3, "#1", "#2" and "#3", both in the same metals, and Season 0 Veteran, a star over crossed swords in
  platinum), and Halloween 2026, a lit jack-o'-lantern in a look of its own: the ribbed pumpkin metal, a
  night-green center and a green ribbon (`STANDALONE_LOOKS` in `titleArt.ts`). A track this build does
  not know is drawn in Conquest's colors, and its steps as bare numbers. **Only the highest rank of each track is ever shown or worn**, so
  the client never filters: it draws what the server sends, in its order.
- **Every title is a medal** (`TitleEmblem`): an SVG per id, drawn by the
  medal rules of [`DESIGN.md`](../DESIGN.md), in the metal `titleArt.ts` gives it
  (`TITLE_METALS`, bronze up to prism) and the colors of its track (`enamelOf`,
  `ribbonOf`). An id this build has no picture for gets the first letter of its
  name in silver, so a new title shows before its art ships. A season's title (`SEASONS` in `titleArt.ts`, the
  season's number) has a frame of its own, so it stands out from every track: a smaller disc in a laurel wreath of
  its metal, a crimson center and ribbon (`--season`), and a scroll across the bottom that reads "S0". The zero is
  drawn narrower than the S, so Luckiest Guy's round 0 does not read as "SO". `locked` greys it
  with a padlock, for a rank not held. Each medal names its masks with `useId`:
  two on one page cannot share one. Its colors are tokens set through `style`,
  since an SVG presentation attribute does not resolve `var()`.
- **The public card** wears the worn title: a banner (`TitleBanner`, the rank line
  and the name), and a ring of its metal inside the card's ink border
  (`title-frame-<metal>` on the `Modal`). Under it, one medal per title shown, with its name. A hover or a press on a
  medal says in a `Bubble` how it was earned (`feat`, the server's words: "Took 1,000 tiles"), as the streak flame
  does. A title the server sends no feat for is not a button.
  OG is also a stamp beside the name (`OgStamp`).
- **The Progress tab** is the worn title (a compact banner) and the stats, then
  three folds (`components/Fold.tsx`, a bar that opens what is under it). Titles
  is open first and holds "Wear a title" (a
  `radiogroup` of the titles that can be worn; a press sends `WearTitle` and reads
  the titles again), and one `TrackPath` per track: every rank, its threshold
  (`stepLabel`), a bar filled up to the progress (`filledOf`), and in its header
  what is left to the next rank (`leftLabel`). The path scrolls sideways and opens
  centred on the next rank. `useTitles` reads `GetTitles` (the click token, as
  `GetProfile`) each time the panel opens and after a run of clicks, at the pace
  of `useMySeason`; a failed read says so. Then **"Fought for" and "Fought
  against"** (`FrontFolds`), closed first: each bar shows its top 3 flags and how
  many countries, and opens on every one of them. Open, a player on 30 countries
  had two lists longer than the screen under the titles. `useFronts` reads
  `GetFronts` (`AccountStore.fronts`) at the same moments, and a failed read
  shows nothing. What is open stays open through these reads, and closes when
  the tab is left. The stats are the player's own card, read by the
  `accountId` the `AccountStore` keeps (`OwnStats`), and shown once the account
  has a username. Tiles taken counts each take at once
  (`useOwnTakes`): `GetPlayer` is cached 10s, so it is read only once.
- **The unlock moment is live.** The player stream carries `titleEarned` to a
  stream opened with this player's token. `useRoster` hands it to `Viewer`, which
  announces it (the announcements queue, in [Battles](battle.md#the-announcements-queue))
  and shows `TitleUnlocked` over the game, one full-screen moment at a time. It is not a
  `Modal` but the whole screen: it goes dark, "New title!" slams in, the medal
  spins in and lands with a flash, a shockwave and confetti, then the name and the
  rank line. The `title` sound is a drum roll that lands on the same beat; both
  read `TITLE_REVEAL` (`domain/titleReveal.ts`). **Nothing closes it before
  `TITLE_REVEAL.ready`**, and a click beside it never does: a player spamming the
  globe would otherwise dismiss it unseen. Then "Close" and "Wear it"
  (`AccountStore.wearTitle`; left out when no account store is wired) appear, and
  Escape works. The server sends only the highest
  rank per track of what one take earned, so a jump of two ranks is one overlay.
  A title earned while no tab is open is never announced; it is simply there next
  time.
- **The worn title follows the name** in the chat, the roster and the season
  board, as a small medal (see [Who is playing](who-is-playing.md)).
- **UI copy is not documentation.** The card and the tab say nothing about the
  rules ("one per track", "others see the title you wear"): what is drawn is the rule.
