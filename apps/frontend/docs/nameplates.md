# Nameplates

**A player's name is drawn from one store, by the account id beside it: the chat's messages, quotes and reactions, the
season board and your own line on it, the Hall of Fame's players and the battle debrief's.** The chat and scoring send
an account id and nothing else of the player; the page reads the name, its color, the crown and whether it is a guest
from player's `GetNameplates`, as it reads the badges from progression's `GetBadges` ([Badges](badges.md)). See the
backend's [Player](../../backend/docs/player.md) for the read (`GetNameplates`). Before 2026-11-06 the chat and scoring
asked player for the names at each read and copied them into their answers.

- `backends/nameplates.ts` — `Nameplate` (`name`, `color`, `admin`, `guest`), `AccountNameplate`, `NameplatesBackend`
  and `MAX_NAMEPLATES_ASKED` (100, as the server). `ConnectPlayerBackend.nameplates` calls
  `player.v1.PlayerService/GetNameplates`, a GET with no token: names are public. The server leaves out every account
  the other players cannot name: unknown, deleted, asked deleted, or never shown.
- `app/nameplates/nameplatesStore.ts` — the store, `useNameplates`, and its hooks: `useNameplate(accountId)` reads one
  account's nameplate, and `useOwnNameplate` writes the page's own username and color. `useNameplatesFeed`, which
  `PlayScreen` calls with the backend, serves the store and empties it once it stops.
- `app/nameplates/nameplate.ts` — what a reader draws of one: `nameOf` (`[deleted]` for an account nobody can name,
  empty until read), `paintOf` (the color for `authorStyle`, grey for a guest or nobody), and `lineOf`, the
  `PlayerLine` a card opens on, none for nobody.
- `app/nameplates/PlayerName.tsx` — a name in its color by account id, masked by the CrazyGames word list. The Hall of
  Fame's players and the debrief's draw it; the chat and the board call `useNameplate` themselves, since they open a
  card from it too.
- `backends/fakeNameplates.ts` — `FakeNameplates`, the fake's `GetNameplates`: the fake account (your own username),
  the fake presence, the fake chat's chatters and the fake board answer, the first that knows an account, so one
  account has one name everywhere. `fakeNameplates.calls` in the console counts the calls.

**How a nameplate is read** (see [Stores](stores.md#a-store-that-reads-by-itself)), as a badge is:

- **Asked, then batched**: what is asked within a frame (`NAMEPLATE_BATCH_MS`, 16 ms) goes out in one call, cut at
  `MAX_NAMEPLATES_ASKED`. A chat history of 200 messages is one call, and most of its authors are already known.
- **Never twice at once**, and **fresh for 5 minutes** (`NAMEPLATE_FRESH_MS`): another player's rename shows within 5
  minutes, or at once while it is online (below).
- **An account left out is kept as `null`**: nobody to name, `[deleted]` in the chat, and not asked again while fresh.
- **A failed read changes nothing on screen** and is tried again no sooner than `NAMEPLATE_RETRY_MS` (10 s): shorter
  than the badges' minute, since a message waits for its name (below).
- **An empty account id is nobody**: a chat message kept before accounts. None is left since the retention passed.

**What the page already knows is written at once**, and a read sent before it is dropped for that account:

- **The roster's lines** (`useRosterFeed`): the whole roster when the stream opens, and each line that joins or changes.
  So everyone online, the chat's live authors most of all, is named with no read, and a rename on the roster shows on
  every message at once.
- **The page's own username and color** (`useOwnNameplate`, from `PlayScreen`): the profile at load, and each `SetName`
  and `SetColor` answer, through the `AccountStore`. A profile does not say whether the account is an admin, so an own
  account the store held nothing of is read once more for its crown. A guest's own name is its roster line, or a read.

**A chat message waits for its names** (`whenRead`, in `useRoomLog`): the history, a catch-up and every live message
show once their author and the author of their quote are read, so no line ever shows a blank name and the toast, the
peek and the sound go with the named line. An author online is known at once. Any other is read in the next frame, and
a read that hangs holds the message `NAMEPLATE_WAIT_MS` (2 s) at most; past that it shows, and its name follows when
read. A reaction's popup names who it can and counts the rest.

**Which reader draws what:**

| Where | Draws | Account id |
|---|---|---|
| `ChatLog`'s `MessageLine` | name, color, crown; the card and the report only for a named account | `message.authorAccountId` |
| `ChatLog`'s `Quote`, `QuoteLine`, the composer's reply bar | name, color | `quote.authorAccountId` |
| `ChatPanel`'s toast and folded peek, `ReportMessageModal` | name, color | `message.authorAccountId` |
| `ReactionBar`'s `ReactionWho` | names | `count.reactorAccountIds` |
| `ChatPanel`'s composer foot, for a guest | name | the own account |
| `PlayerStandings`' `StandingRow` and your own line | name, color, card | `standing.accountId` |
| `SeasonStats`' `PlayerRows` (Hall of Fame), `BattleDebrief`'s `Players` | name, color | `row.accountId` |

**Left as they were:** the roster (`PlayersPanel`) draws its own lines, which are player's; the tile's holder and the
territory's players (`HolderCard`, `TerritoryCard`) name an account from its `GetPlayer` card, which they read anyway;
a mute line names the account the chat named when it was muted, kept in the announcement.

**The guest rule is the nameplate's**: a guest's addresses are not linked in the chat (`messageParts`), where it once
read the `guest_` prefix of the name.
