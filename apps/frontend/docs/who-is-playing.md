# Who is playing

The "Online · N" tab beside Chat, in the chat's header, lists
everyone playing: players with a username, then guests, each with a flag, a name
in its chat colour (`authorStyle`, the same hue as in the chat). A line's name
is the one the chat shows for that account: the username, or `guest_` and its
code. No address, and no hash of one, is on it.

- `backends/player.ts` — `PresenceBackend`, `Presence`, `PlayerLine` (what a
  card needs), `RosterEntry` (a `PlayerLine` with its `key`), `RosterEvent`, and
  `PlayerInfoBackend` with `PlayerInfo`. `ConnectPlayerBackend` implements it over
  `player.v1.PlayerService/Announce`, `Leave` and `ListenForEvents`;
  `fakePresenceBackend.ts` is the dev stand-in, with players coming and going on
  their own shifts, diffed into live events every second.
- `domain/presence.ts` — `PresenceSchedule`, when to announce. No clock and no
  network, so every rule is under test. `domain/roster.ts` applies one live event
  (`applyRosterEvent`) and splits the roster into the two groups.
- `app/players/` — `usePresence`, `useRoster` and `usePlayerInfo`, thin hooks
  over the above, `PlayersPanel` and `PlayerCard`.

**A name wears its color and its streak** everywhere it is drawn: the chat log,
the folded quote, the roster and the card. Both come from the server with the
name (`ChatMessage.authorColor` and `authorStreak`, `RosterEntry.color` and
`streak`, `PlayerInfo.color`), read from the account when shown, so a new pick
shows on everything its player ever said once the chat is read again. The
flame (`StreakFlame`) is the Noto fire of the reactions, `role="img"` named
"12-day streak", and is left out under 3 days (`streakShown`). It is a button: a
press, or a mouse resting on it, says "Played 12 days in a row" in a `Bubble`,
which goes when the log under it scrolls. **A guest has
neither**: the server sends it no color and a streak of 0, so a signed-in player
shows a flame only once it has a username.

**A name wears its title too**, in the chat log, the roster and the season board: the medal of the
title its player wears, small, after the crown (`TitleBadge`, a `TitleEmblem`
with `role="img"` named by the title). The server sends it with the name
(`ChatMessage.authorTitle`, `RosterEntry.wornTitle`, `Standing.wornTitle`), read when shown, so a new
pick shows on the roster at once, on the next message, and on older ones once the
chat is read again. A guest
wears none. `backends/title.ts` holds `PlayerTitle` and `titleOf`, the one
decoder of `player.v1.Title`, shared by the chat, the player and the standings backends.

**An admin of the game wears a crown** (`AdminCrown`, gold, `role="img"` named
"Admin") beside its name in the chat log, the roster and the card's title.
The server says so: `ChatMessage.authorAdmin`, `RosterEntry.admin` and
`PlayerInfo.admin`. The card crowns from what was clicked, and from the read
once it lands. In fake mode, Ana is the admin.

**A name opens a player card**, in the roster, on a chat message and on the
season's board (`app/players/PlayerCard.tsx`, a `Modal`). **The card is read by
the line's `accountId`, never by its name** (`PlayerInfoBackend.playerInfo`,
`GetPlayerRequest.account_id`): the name is what the card shows, and a name
that changed hands would open another player's card. The card can report
its player: see [Support](support.md). `Viewer` holds the one
card open and hands `onOpenPlayer` to `ChatPanel` → `ChatLog` and `PlayersPanel`,
and to the board's `BoardStandings`;
without a `PlayerInfoBackend` wired the names are plain text. The card shows the
flag and the country, then, for a player with a username, what
`player.v1.PlayerService/GetPlayer` answers: the title it wears and the titles
it shows (see [Titles](titles.md)), then tiles taken, the current and best
streak, and "Playing since", the day the account was made (left out when the
server does not know it). Under them, **"Fought for" and "Fought against"**
(`PlayerFronts`): the countries the player took tiles for, and the ones it took
them from, each with a bar against the side's first country. Past tense on
purpose: "You're playing for" at the top of the menu is the flag now, and these
are the player's whole history. The card shows
the top 3 of each, all `GetPlayer` sends; the player's own Progress tab shows
every one (below). A side with no tile is left out. They
are the player's total, not the season's (see the backend's [Player](../../backend/docs/player.md)),
so the top one explains the flag beside its name on the Players board. Two
columns on a desktop, which is why the card is 440px wide; one under the other
on a phone. The fake gives its players titles and countries of its own over
their fake stats and creation date. **A guest's card asks nothing**: `GetPlayer`
answers a guest with its name alone (`PlayerInfo.guest`), which the line already
holds, so the card shows the name and the flag alone. It says nothing to the viewer, who may well be signed in. The chat tells
a guest by `GUEST_PREFIX`, which no username starts with: that is a class of
names the server reserves, not one account. **A line with no account id asks
nothing either** (a message kept before accounts, or of a deleted account) and
reads "This player is gone." `GetPlayer` needs no
token and goes out as a GET, like `GetRoster`; `NotFound` (the account is
gone) reads as `undefined`, and shows the same line. `usePlayerInfo` reads it
once per account. `Viewer` keys the card by the account id.
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

**The roster is streamed**, over `ListenForEvents` through `openStream`, with
the token already held (`held()`, never a mint) when there is one. The roster
needs none; the token is what lets the same stream bring this player's own
titles (`titleEarned`, see [Titles](titles.md)). The server reads it when the
stream opens, so `useRoster` checks `heldSession()` every `SETTLE_MS` and
reopens the stream when it changes, keeping the list it has. Every connection starts with the whole roster, then
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
