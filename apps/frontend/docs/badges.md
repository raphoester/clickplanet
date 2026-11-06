# Badges

**A player's badges are its streak and the title it wears, drawn beside its name wherever the name is: the chat
log, the roster, the season board and your own line on it, a tile's holder, a territory's players, the player
card, the Hall of Fame's players and the battle debrief's.** The page composes them: the name comes from the
nameplates store ([Nameplates](nameplates.md)), or the roster line or the card that shows it, and the badges from one
store, by the account id beside the name. Nothing else carries them. See the backend's [Progression](../../backend/docs/progression.md#the-badges) for `GetBadges`.

- `backends/badges.ts` — `Badge` (`streak`, `wornTitle`), `AccountBadge`, `BadgesBackend` and `MAX_BADGES_ASKED`
  (100, as the server). `ConnectProgressionBackend.badges` calls `progression.v1.ProgressionService/GetBadges`, a
  GET with no token: names and badges are public. The server leaves out every account with nothing to show (unknown, a guest,
  deleted or asked deleted, or with neither a streak nor a title).
- `app/badges/badgesStore.ts` — the store, `useBadges`, and its hooks: `useBadge(accountId)` reads one account's
  badge (a plain value, the store's own object), and `useOwnStreak` writes the streak the page read of its own
  account. `WornTitle` and `Streak` draw one each (a `TitleBadge`, a `StreakFlame`); a reader that draws them by
  hand calls `useBadge`. `useBadgesFeed`, which `Viewer` calls with the backend, serves the store and empties it
  once it stops.
- `backends/fakeBadges.ts` — `FakeBadges`, the fake's `GetBadges`: the fake presence (its roster, its card and the
  chat's chatters, which are the same players) answers first, then the fake board for the players only it has, so
  one account wears one badge everywhere.

**How a badge is read** (see [Stores](stores.md#a-store-that-reads-by-itself)):

- **Asked, then batched.** `useBadge` asks the store after every render. The accounts asked within a frame
  (`BADGE_BATCH_MS`, 16 ms) go out in one call, cut into calls of `MAX_BADGES_ASKED`. A chat history of 200
  messages from 40 authors is one call; the roster, the board and each card add their own few.
- **Never twice at once.** An account being read, or asked already in this frame, is not asked again.
- **Fresh for 5 minutes** (`BADGE_FRESH_MS`). A badge older than that is read again the next time a name shows it.
  So another player's new title, or a streak it extended today, shows within 5 minutes, at the next render of a
  name that shows it.
- **An account left out is kept as `NO_BADGE`**, nothing to draw, and is not asked again while fresh.
- **A failed read shows nothing new and breaks nothing.** The name draws as it would; a badge read before stays,
  and an account never read has none. The accounts of that call are read again no sooner than `BADGE_RETRY_MS` (1
  minute), at their next render. So a backend that does not serve `GetBadges` yet (`Unimplemented`) costs one call
  a minute, and the page shows no badge.
- **An empty account id has none**: a message kept before accounts, or of a deleted account.
- **A badge read again the same keeps its object**, so a name showing it does not render.

**What the page learns of its own account it writes at once**, before any read, and a read sent before cannot undo
it:

- **The title worn**: `AccountStore.wearTitle`, from `WearTitle`'s answer (the Progress tab, the profile and "Wear
  it" on the unlock moment), and `AccountStore.titles`, from `GetTitles`' `worn`, which follows a title earned. Not
  for an account changed meanwhile (`generation`).
- **The streak**: the profile's and the Progress tab's own stats (`OwnStats`, `GetProgress` on the own account), with
  `useOwnStreak`.
- **A title earned** (`titleEarned` on progression's stream, see [Titles](titles.md)) may change the title worn
  (`wearing.WornOf` on the server: a higher rank on the worn track replaces it), which the page cannot tell, so
  `Viewer` expires its own badge and reads it again at once.
- **A player card** reads the badge of its player again as it opens (`expire`), so the title on its banner agrees
  with the titles `GetProgress` lists under it.

**Which reader draws what, and by which id:**

| Where | Draws | Account id |
|---|---|---|
| `ChatLog`'s `MessageLine` | title, streak | `message.authorAccountId` |
| `PlayersPanel` | title, streak | `entry.accountId` |
| `HolderCard` | title (48px), streak | `holder.line.accountId` |
| `TerritoryCard`'s `PlayerName` | streak | `line.accountId` |
| `PlayerStandings`' `StandingRow` | title | `standing.accountId`; your own line, `caller.accountId` |
| `PlayerCard` | the banner and the frame | `player.accountId` |
| `SeasonStats`' `PlayerRows` (Hall of Fame) | title | `row.accountId` |
| `BattleDebrief`'s `Players` | title | `player.accountId` |

The title the season board, the Hall of Fame and the debrief draw is the one its player wears **now**: scoring
never kept a title, it copied the one worn when it was read. The profile's own medal and its Progress tab are
`GetTitles`' (`useTitles`), not the store's.
