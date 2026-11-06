# Who is playing

The "Online · N" tab beside Chat (beside World and the flag's room when the
player holds a flag; see [Rooms](chat.md#rooms)), in the chat's header, lists
everyone playing: players with a username, then guests, each with a flag, a name
in its chat colour (`authorStyle`, the same hue as in the chat). A line's name
is the one the chat shows for that account: the username, or `guest_` and its
code. No address, and no hash of one, is on it.

- `backends/player.ts` — `PresenceBackend`, `Presence`, `PlayerLine` (what a
  card needs), `RosterEntry` (a `PlayerLine` with its `key`), `RosterEvent`, and
  `PlayerInfoBackend` with `PlayerInfo`, the public card: `Player`, who the player
  is (`PlayerReader`), and `Progress`, how far it went (`ProgressReader`, in
  `progression.ts`). `ConnectPlayerBackend` implements the presence over
  `player.v1.PlayerService/Announce`, `Leave` and `ListenForEvents`, and
  `PlayerReader` over `GetPlayer`; `PlayerCards` (`playerCards.ts`) is the
  `PlayerInfoBackend`, over a `PlayerReader` and a `ProgressReader`;
  `fakePresenceBackend.ts` is the dev stand-in, with players coming and going on
  their own shifts, diffed into live events every second.
- `domain/presence.ts` — `PresenceSchedule`, when to announce. No clock and no
  network, so every rule is under test. `domain/roster.ts` applies one live event
  (`applyRosterEvent`) and splits the roster into the two groups.
- `app/players/` — `usePresence`, `useRosterFeed` and `usePlayerInfo`, thin hooks
  over the above, the roster's store `useRoster`, `PlayersPanel`, `PlayerCard`,
  and `usePlayerCard` with `PlayerCardPlace`, the one card open.

**A name wears its color** everywhere it is drawn: the chat log, the folded quote, the roster and the card. It
comes with the name: the nameplate's (`Nameplate.color`, see [Nameplates](nameplates.md)), the roster's
(`RosterEntry.color`) or the card's (`PlayerInfo.color`), each player's own now, so a new pick shows on everything its
player ever said once its nameplate is read again, and at once while it is online.

**A name wears its badges, its streak and its title**, in the chat log, the roster, the season board and the
cards. They do not come with the name: each is read from the badges store by the line's `accountId` (see
[Badges](badges.md)), so another player's new title or longer streak shows within 5 minutes, and your own at once.
The flame (`StreakFlame`, drawn by `Streak`) is the Noto fire of the reactions, `role="img"` named "12-day streak",
and is left out under 3 days (`streakShown`). It is a button: a press, or a mouse resting on it, says "Played 12
days in a row" in a `Bubble`, which goes when the log under it scrolls. The title is the medal of the title its
player wears, small, after the crown (`TitleBadge`, drawn by `WornTitle`, a `TitleEmblem` with `role="img"` named
by the title). **A guest has neither**, nor a color: the server gives a guest no badge, so a signed-in player shows
a flame and a medal only once it has a username. `backends/title.ts` holds `PlayerTitle` and `titleOf`, the one
decoder of `progression.v1.Title`.

**An admin of the game wears a crown** (`AdminCrown`, gold, `role="img"` named
"Admin") beside its name in the chat log, the roster and the card's title.
The server says so: `Nameplate.admin`, `RosterEntry.admin` and
`PlayerInfo.admin`. The card crowns from what was clicked, and from the read
once it lands. In fake mode, Ana is the admin.

**A name opens a player card**, in the roster, on a chat message and on the
season's board (`app/players/PlayerCard.tsx`, a `Modal`). **The card is read by
the line's `accountId`, never by its name** (`PlayerInfoBackend.playerInfo`,
`GetPlayerRequest.account_id`): the name is what the card shows, and a name
that changed hands would open another player's card. The card can report
its player: see [Support](support.md). **The one card open is a store**,
`usePlayerCard` (`app/players/playerCardStore.ts`, see [Stores](stores.md)): `ChatLog`,
`PlayersPanel`, the board's `PlayerStandings`, `HolderCard` and `TerritoryCard` open a
player in it themselves, and the last two close their own card first. `PlayerCardPlace`
shows the card. `PlayScreen` renders it only with a `PlayerInfoBackend` wired, and the
store is `openable` only while it is there: without it the names are plain text. The card shows the
flag and the country, then, for a player with a username, what `PlayerCards` reads: who
it is from `player.v1.PlayerService/GetPlayer`, and from
`progression.v1.ProgressionService/GetProgress` the titles it shows, under the title it wears, which is its badge
(see [Titles](titles.md) and [Badges](badges.md)), then tiles taken, the current and best
streak, and "Playing since", the day the account was made (left out when the
server does not know it). Under them, **"Fought for" and "Fought against"**
(`PlayerFronts`): the countries the player took tiles for, and the ones it took
them from, each with a bar against the side's first country. Past tense on
purpose: "You're playing for" at the top of the menu is the flag now, and these
are the player's whole history. The card shows
the top 3 of each, all `GetProgress` sends; the player's own Progress tab shows
every one (below). A side with no tile is left out. They
are the player's total, not the season's (see the backend's [Progression](../../backend/docs/progression.md#the-fronts)),
so the top one explains the flag beside its name on the Players board. Two
columns on a desktop, which is why the card is 440px wide; one under the other
on a phone. The fake gives its players titles and countries of its own over
their fake stats and creation date. **A guest's card asks nothing**: `GetPlayer`
answers a guest with its name alone (`PlayerInfo.guest`), which the line already
holds, so the card shows the name and the flag alone. Where a card is read for an
account not known as a guest (a tile's holder), `PlayerCards` shows a guest with no
progress, whatever `GetProgress` says. It says nothing to the viewer, who may well be signed in. The chat tells
a guest by `GUEST_PREFIX`, which no username starts with: that is a class of
names the server reserves, not one account. **A line with no account id asks
nothing either** (a message kept before accounts, or of a deleted account) and
reads "This player is gone." `GetPlayer` and
`GetProgress` need no token and go out as GETs, side by side, each retried while
the server cannot be reached. A `NotFound` from `GetPlayer` (the account is gone)
reads as `undefined`, and shows the same line. A `NotFound` from `GetProgress` alone
shows zeros, and no title and no front. `usePlayerInfo` reads it
once per account. `PlayerCardPlace` keys the card by the account id.
Escape closes the card and leaves the roster open: `useEscape` does nothing
while a modal dialog is on the page.

**It announces only with a token it already holds.** `SessionProvider.held()`
answers the click token in hand and never mints: a mint is a Turnstile check,
and presence is not worth one. So a visitor who has never clicked is not listed,
by design; one whose kept token is still live is listed from the load — see
[Sessions](sessions.md). The schedule announces as soon as a token is held that the last
announce did not go out under (the first click, a re-mint, a sign-in), once
the flag or the username has held still for a second, and every
30s — the server drops a player 90s after its last one. An announce carries the
flag alone: the server reads the name off the account. An announce refused
`unauthenticated` drops the token and is **not** retried with a fresh one, which
would mint; the next click brings one. `NoSession` holds nothing, so a build
without a sitekey never announces.

**The roster is a store**, `useRoster` (`app/players/rosterStore.ts`, see
[Stores](stores.md)): loading, unavailable, or the list. `useRosterFeed`, which
`PlayScreen` calls with the `PresenceBackend`, follows the stream into it, and lists
nobody again once it stops. `PlayersPanel` reads the list itself and the chat's
tab reads the count (`onlineCount`), so a renamed line renders the list alone.
`PlayScreen` reads only the count, which it gives the globe (`crowd`, see
[Ghosts](ghosts.md)).

**The roster is streamed**, over `ListenForEvents` through `openStream`, with no
token: the roster needs none, and the titles earned have a stream of their own (see
[Titles](titles.md)). Every connection starts with the whole roster, then
sends one line that joined or changed (`entry`) or one key that left (`left`).
A line is named by its `key`, which the server keeps through a new flag, a
sign-in and a new name, so a guest who signs in is one row renamed in place;
rows are keyed by it in React too. `applyRosterEvent` puts a changed line where
the server's sort would (`compareRosterEntries`); a rare disagreement about
case in a non-ASCII name lasts until the next reconnect's whole roster. While
the stream is down the last list stays. A 404 (read as `unimplemented`) calls
`onUnavailable` once and stops the stream for good, which hides the button, as
does a build with no presence backend wired. `GetRoster` is no longer called.

**A closing page says it left.** `usePresence` calls `leave` on `pagehide`,
unless the page is only kept in the back-forward cache (`persisted`). `Leave`
carries the held token, never a fresh one, and goes out on a second player
client whose `fetch` sets `keepalive` (`newKeepalivePlayerServiceClient`), so it
is still sent after the page is gone. The server takes the account off at once.
Two tabs of one browser are one account: closing one takes the line off until
the other's next announce, at most 30s later.
