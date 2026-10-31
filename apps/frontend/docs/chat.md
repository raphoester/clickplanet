# Live chat

The client for the backend's second bounded context: `chat.ts` declares
`ChatSender`, `ChatHistoryGetter`, `ChatListener`, `ChatReactor` and
`ChatSeenMarker` (plus `ChatBackend`, the five together), `chatBackend.ts`
implements them against `/chat.v1.ChatService/` alone — `SendMessage`,
`GetHistory`, `React`, `MarkSeen` and the `ListenForEvents` stream — and
`fakeChatBackend.ts` is the dev stand-in. On a desktop `ChatPanel` is the
right-hand column, folded and unfolded from its own header. On a phone it is the
Chat tab's sheet, and `Viewer` holds whether it is open (`open`,
`onOpenChange`); see [The screen](screen.md). **It is always
mounted**, open or not, on both: it owns the history load, the stream, the unread
count and the sound. **A resumed stream reads the history again**: `addMessages`
takes what it missed, and `withNewerReactions` the reactions that moved meanwhile. Closed on a phone it draws only the peek (below) and hands
the unread count up through `onUnread`, for the tab's badge.

**The chat and who is online share the panel**: with a roster wired, its header
is two tabs, Chat and "Online · N" (see [Who is playing](who-is-playing.md)).
While the list is shown, the Chat tab wears the unread badge: the log is not on
screen, so nothing is marked seen, and the open panel has no other badge.

**A Discord bar sits over the log** (`DiscordBar`), on a desktop and in the
phone's sheet: a link to `DISCORD_INVITE`, then TikTok, YouTube and Instagram
as three icons, and a cross. **The link says "Join Discord", not "Join the
Discord"**: beside three icons the longer label no longer fit on a 375px phone
or in the panel's default width, and a cut label looks broken. The cross hides all of it for
`DISCORD_BAR_RETURNS_AFTER_MS` (20 hours), from the time of the press kept in
`clickplanet-discord-bar` (`discordBar.ts`, `useDiscordBar`). **20 hours and
not 24**, so a player who comes back at the same hour each day finds it again
each day. It is read when the chat is drawn, so it comes back on the next
visit, or the next time the chat is opened, and never under a player who is
reading. **It is a plain link**: the privacy page says nothing is sent to
Discord before a click, so it asks Discord for no member count.

**The open panel is resized from its top edge, its left edge or its top-left
corner**, and a double-click on one puts the default back (`useChatSize`). Not
on a phone, where it is a sheet. What the player dragged to is kept
in `clickplanet-chat-size` and written on `:root` as `--chat-wanted-width` and
`--chat-wanted-height`; `index.css` clamps them into `--chat-width` and
`--chat-height`. **The clamp keeps the chat 16px clear of the dock**, centred at
the bottom: the width stops at `50vw - var(--dock-width) / 2 - 32px`, and under
1100px wide, where that leaves too little, the chat sits above the dock instead
(`--chat-lift`). The log stays pinned
to its newest line while the panel changes size (a `ResizeObserver` in `ChatLog`).

**`MAX_TEXT_LENGTH` in `chat.ts` mirrors `chat.service.maxTextLength` on the
backend**, counted in code points as the server counts runes. It is the
composer's bound, not a defence — the server sanitizes and rejects on its own.

**On the CrazyGames page the words of CrazyGames' list are masked** in the text, the quotes and every name, as they
are drawn: see [The word filter](crazygames.md#the-word-filter).

**Message text is rendered as text, never as HTML.** The backend stores it raw
and says so; React escaping is what makes that safe, so never reach for
`dangerouslySetInnerHTML` here.

**An address in a message is a link** (`domain/chatLinks.ts`): `messageParts`
cuts the text into text and links, and `ChatLog` draws each link as an `<a>`
that opens a new tab with `rel="noopener noreferrer nofollow ugc"`.

- **Only `http://` and `https://`, written in full.** A bare `clickplanet.lol`
  stays text: telling a domain from two words and a dot needs a list of TLDs.
- **A guest's address stays text.** A guest account is free, and a private tab
  is a new one, so a mute does not stop a guest who posts links.
- **An address with a user name is not linked**: `https://clickplanet.lol@evil.example`
  reads as this site and opens the other one.
- The punctuation that ends a sentence, and a bracket the address did not open,
  are left out of the link.
- **Only the balloon links.** The quote of a reply, the folded header's line and
  the phone's peek are buttons, so they show the text.

**The server names every sender; the client sends no name.** A player with a
username posts under it. Every other account is a guest, shown as `guest_` and a
6-hex code the server drew once for that account (`guest_a1b2c3`, `GUEST_PREFIX`)
— no username starts with the prefix, so a guest cannot pass for a player, and
no two accounts share a code. **A name is one account at one moment only**: a
rename frees it and another account can take it, so nothing here tells a
player by its name. `ChatMessage.authorAccountId` does (empty for a message
kept before accounts, and for a deleted account). **Nothing about the
sender's address is public**: the `#tag` beside every name is gone from the wire.
The composer asks nobody for a name; its foot reads "as <username>", or for a
guest "as a guest" until its first message comes back, then the name the server
gave it (`nameSentUnder`). The client still keeps a UUID in
`clickplanet-chat-identity` (`chatIdentity.ts`, `useChatIdentity.ts`, held by
`ChatPanel`) because `SendMessageRequest.author_id` carries it; the server
trusts it for nothing, and a name stored there by an older build is dropped.

**`SendMessage` and `React` always carry the click token**, guests included:
the server refuses a caller with no account (`unauthenticated`). So
`ChatServiceBackend` calls `token()`, which mints when none is held, as a click
does — chatting before the first click costs a Turnstile check. A token the
server refuses is dropped and the call made once more with a fresh one (a
refused call posted nothing, so this cannot post twice); a second refusal, or a
mint that failed, is `ChatNoSessionError`, and nothing is sent. `Viewer` reads
the username and the account id off the `AccountStore` and hands both to
`ChatPanel`: the name for the composer's foot, the id to tell the player's own
messages. **A message is yours when its id is in `mine`, or its
`authorAccountId` is your account** (`useChat`'s `own`): the id the
`AccountStore` read from the profile of a signed-in player, or, for a guest,
the account of the last message it sent (`accountSentUnder`). The badge, the
peek and the sound all ask `own`. Never the name: another account can hold the
name you left, and its messages are not yours. A guest's very first message
can ping if its broadcast beats the answer.

**`sendMessage` is the one call that is not wrapped in `retrying`.** A retry
after a connection dropped mid-request would post the message twice, visibly, to
everyone; a message the player can retype is the cheaper failure. `getHistory`
is retried like every other read.

The refusals map to their own error classes and are reported **inline in the
composer, not as a modal** — unlike a refused click, the text is still in the box
and the advice is one line:

- `resource_exhausted` → `ChatRateLimitedError`, chat's own bucket (one message
  every 3s), unrelated to the click bucket
- `permission_denied` → `ChatBlockedError`, the address is in `chat.blockedIPs`
- `permission_denied` with a `MuteRefusal` detail → `ChatMutedError`, an operator
  muted the account or its network; the composer says until when (the time, or
  the date and time past 20 hours). A muted reaction is undone and says the same
- `invalid_argument` → `ChatRejectedError`, the server refused the content
- `not_found` → `ChatMessageGoneError`, the message a reply answers is no longer
  shown (see [Replies](#replies))
- `unauthenticated` twice, or no token to be had → `ChatNoSessionError`

The server always runs chat, so there is no "chat is off" error. **A history that
cannot be loaded hides the panel entirely** rather than showing a broken box:
`useChat` goes to `unavailable` and `ChatPanel` renders nothing. So does a build
with no chat backend wired.

The composer **clears the box when the send starts, not when it lands**, and puts
the text back only if the box is still empty when a refusal comes in. Clearing on
success instead wipes whatever was typed while the message was in flight, which
is exactly what a fast typer does.

## Replies

A message can answer another one. **The reply is a message in the one thread**,
with the beginning of what it answers in a small bubble above its balloon
(`ChatMessage.replyTo`, a `ChatQuote`: the id, the author's name and color, the
text). The server sends the quote on the message, so a reply still shows it once
the log has dropped the message it answers.

- **A button beside each balloon starts one** (`.chat-reply-add`, next to the
  reaction button and shown like it). `ChatPanel` holds the message being
  answered (`replyingTo`), the composer shows it above the box (`.chat-replying`)
  and takes the focus, and `OutgoingMessage.replyTo` carries its id. The cross
  and Escape drop it (`useEscape`, so on a phone Escape drops the reply before it
  closes the sheet).
- **`QuoteLine` draws both**, the bubble in the log and the bar over the box. It
  cuts the text at `QUOTE_MAX_LENGTH` (100) with `truncate`, and the CSS cuts the
  line where it ends. The name is in its author's color: `.chat-quote` and
  `.chat-replying` carry `--author-hue` like a message.
- **One level**: a reply to a reply shows the reply it answers, never a chain.
- **A reply opens a group** (`startsGroup`), so its head says who answers even in
  the middle of its author's run. So does another account, and the same
  account under a new name: a run is one `authorAccountId` under one name.
- **A quote goes to its message** when the log holds it: a press scrolls the
  message to the middle and lights its balloon for `SPOTLIGHT_MS` (`jumpTo` in
  `ChatLog`, which finds the line by `data-message`). With the message gone from
  the log the quote is plain text.
- **The reply goes when the send starts and comes back with the text if the
  message is refused**, like the box. A reply to a message the server no longer
  shows is `ChatMessageGoneError`: the composer says so and the reply is dropped,
  so the text can go out as a plain message.
- In fake mode one opening line is a reply, and every third line after it
  answers a random earlier one.

## Removed messages

**An operator can remove a message** (the back office, on a report). It keeps its
place and its author, and reads "This message has been deleted" (`REMOVED_TEXT`
in `QuoteLine.tsx`), in italics. So does the quote of every reply to it.

- **The server never sends the text again**: the history answers `removed` with
  no text and no reactions, on the message and on a quote.
- **The stream sends `message_removed`** with its id, and `useChat` empties the
  message and every quote of it at once (`withRemoved` in `domain/chatLog.ts`).
- **Nothing more can be done with it**: no reaction, no reply, no report.
- A removal missed while the stream was down shows until the next page load: a
  catch-up only adds what the history holds.
- In fake mode `fakeChat.remove()` in the console removes the newest line, and
  `fakeChat.remove(id)` a given one.

## Reactions

A message carries reactions from a fixed set, `chat.v1.Reaction`: the proto enum
is the list, and the backend refuses any other.

- **Drawn from our own images, never the system's emoji font**, which looks
  different, or broken, on every platform (Windows most of all). They are
  Google's Noto Emoji (Apache 2.0), vendored by `npm run reactions`
  (`scripts/generateReactions.mjs`) from a pinned commit into
  `static/reactions/` under content-addressed names, with
  `app/chat/reactionsAsset.ts` generated beside them. **A new reaction** is a
  value at the end of the proto enum, a line in the script's `REACTIONS`, and a
  run of the script. A reaction this build has no image for is not shown.
- `ChatLog` shows the counts under each balloon (`ReactionBar`), and a button
  beside the balloon (`AddReactionButton`) that opens the picker. The button
  shows on hover; a touch screen has no hover, so there it stays, faint. The
  picker closes on a pick, on Escape and on a click elsewhere.
- **A chip says who reacted while the pointer rests on it**, or while it holds
  the focus: the reaction's name, then the players under it (`whoReacted`,
  `ReactionWho`). The names come from the server on the count itself
  (`ReactionCount.reactors`), oldest first, cut at 20; the server reads them
  from the accounts under the reaction rather than any stored name, so a rename
  shows here too. `count` is what says how many gave it, so the popup ends on
  "and N more" whenever it has fewer names than that — a long list the server
  cut, or somebody it could not name at all. It is drawn in a portal on the body, not beside the chip: the log
  both scrolls and clips. Anything that
  moves the chip — a scroll, a resize — closes it rather than making it follow.
- **`mine` is only known from a call.** `GetHistory` sends the reader's token
  (`SessionProvider.identity()`, never a mint) so the server can mark the
  player's own; `React` answers the counts with `mine` set. The stream is
  nobody's, so `mergedReactions` keeps what the log already knew. A player
  whose click token is not held yet when the history loads is named by its
  identity token instead (see [Sessions](sessions.md)); one with neither sees its own reactions
  unmarked; the server treats a second "on" as nothing, so a click still ends
  right.
- `React` goes out with the click token, minted when none is held, like a
  message: every caller reacts as its account, guests included.
- **Each message keeps its reactions' version** (`reactionsVersion`). The
  server publishes tallies with no lock, so two frames can arrive in the wrong
  order: `applyReactionsChange` (a frame) and `applyReactionsAnswer` (the answer
  to this player's own reaction) drop one older than what the log holds.
- `useChat.react` shows the change at once (`toggledReactions`), then takes the
  server's answer, or undoes it when refused. A message the server no longer
  shows reads as `ChatMessageGoneError`. It puts this player's own name in and
  out of the popup's list too: `useChat` derives `displayName` — the `username`
  prop, or the name the server posted under (`nameSentUnder`) for a guest — and
  reads it through a ref, so a guest learning its name builds no new callback.
  Without one the reaction is counted and nobody new is named, until the
  answer lands.
- A reaction is not a new message: `ChatLog` shows the "New messages" pill only
  when the last message changes, the unread count counts messages and
  announcements, and the sound only messages.

## Announcements

The chat also shows lines nobody sent: `ChatEvent.announcement` on the stream,
and `GetHistoryResponse.announcements` beside the messages. Five kinds today:
`bomb`, every bomb that went off, `mute`, every mute an operator gave,
`round`, every round of a season that closed, `fortify`, every landmass of 50
tiles or more a flag fortified, and `champion`, the country that won a season.

- **Decoded, not trusted**: `decodedAnnouncement` reads the `kind` and parses
  the JSON `payload` into a typed `ChatAnnouncement`. A kind this build does not
  know, or a payload that is not the kind's, is dropped, so the server can ship
  a new kind first.
- **The payload is values, the client writes the sentence**: a bomb line is
  `describeBlast`, the same words as `BombNews`, so the chat and the news line
  never disagree. Both name the landmass with `landmassName`, as a fortify line
  does, not the country it belongs to: a bomb on Tasmania says "Tasmania", not
  "Australia". A bomb told before the payload had a `landmass` names the
  country. A mute line is the name and `describeMute` of its seconds
  (`domain/mute.ts`): "one hour", "90 minutes", "2 days".
- **Kept apart from the messages** (`useChat`'s `announcements`,
  `addAnnouncements`) and put in one list only to draw (`interleave`, by time).
  So a burst of bombs never pushes a message out of the log, and the sound and
  the "New messages" pill count messages alone. The unread count counts
  announcements too: a bomb while away is something missed.
  **Once the message log is full, `interleave` leaves out every announcement
  older than its oldest message**: the two logs are capped apart, so in a long
  session the older bombs piled up on top of the chat. The server does the same
  for the history.
- **Not a balloon**: `ChatLog` draws a centred line (`.chat-announcement`) with
  the bomber's flag and the time. It ends the run above it, so the next message
  says again who is talking.
- **A round is a gold card** (`RoundLine`, `.chat-announcement-round`): "Day 5
  is over!" (`roundOverLine`, the words of the reveal), the time, then one place
  per podium country: its medal, its flag, its name and the points it won. **A
  battle's round of its own is told as the battle's, never as a day**: ⚔️ for 🏁,
  "The battle is over!" or "The Final Battle is over!", and its times in a war chip
  ("×3"), since a round of its own is numbered apart from the days. The payload is
  `{number, finale, own, times, podium}`, `own` and `times` absent for a day (read
  as a day, ×1); the podium is the places ranked 3rd or better, ties included (the
  backend's [Announcements](../../backend/docs/chat.md#announcements)). A place
  with no country, or with a rank or points under 1, is dropped. Its time is the
  round's end.
- **A champion is a gold line** (`ChampionLine`, `.chat-announcement-champion`):
  🏆, the flag of each champion and "Germany wins Season 0" (`winsLine`, the
  words of the reveal). The payload is `{season, countries}`; a champion with
  no country is dropped. Its time is the season's end. The server keeps one a
  season, so it shows once.
- In fake mode `main.tsx` hands every `FakeBackend` bomb to
  `FakeChatBackend.announceBomb`, with no ground: the fake has no borders.
  `closeDay()`, `closeBattle()` and `closeFinale()` announce their round through
  `FakeChatBackend.announceRound`, and `closeFinale()` its champion through
  `FakeChatBackend.announceChampion`, once a season as the server does.
  `fakeChat.mute()` in the console announces a mute of the player for an hour
  and refuses its posts and reactions until it ends; `fakeChat.mute(600, "Ana")`
  only announces somebody else's.

## Saying that a message landed

The globe is what the player is looking at, so an arriving message has to catch
the eye without stealing it. Four things say it, each for a different glance:

- **A message lights up as it comes into view.** `ChatPanel` holds the ids in
  `flashing` for `FLASH_MS`, matched to the `chat-message-glow` animation,
  and `ChatLog` turns that into a class. What lights up is what `idsSince`
  reports as unseen, so the same rule covers one message arriving into an open
  panel and a whole backlog the moment a folded one is unfolded.
- **Your own message never flashes.** `useChat` keeps the ids `sendMessage`
  returned in `mine` and `ChatPanel` filters them out — you know you sent it.
  This is the only reason `mine` exists.
- **A folded panel breathes.** `chat-waiting` puts the accent on the title, pops
  the unread badge (remounted on every count change, so it replays per message)
  and pulses the panel's own border and glow. It is a slow breath rather than a
  blink: this sits over a game. On a phone the badge is on the Chat tab.
- **The newest line is quoted under the folded header**, in its author's colour.
  It is `aria-hidden` — a screen reader gets the count from the badge and the
  text from the log, and the quote would only say it a third time. **On a phone
  it is a peek over the dock instead** (`.chat-toast`): the newest line, as a
  balloon, for `TOAST_MS` (4s) after it lands, and a press opens the chat. It is
  a button named by the line it quotes, since it is the only thing on screen
  that opens the chat from it.

## What was missed since the last visit

**The server keeps until when each account saw the chat**, so the badge on
arrival counts what was said while the player was away: messages and
announcements, not the player's own messages. See the backend's
[Seen](../../backend/docs/chat.md#seen).

- **A time, not an id.** `GetHistory` answers `seenUntil`; `useChat` hands
  `ChatPanel` the time to count from (`seenAtLoad`), and `unseenAfter` counts
  the lines after it, open or not. A history with no mark is a first visit:
  nothing counts, and the newest line of the history is sent as the mark, so
  the next visit has one even if the chat is never opened in this one.
- **Opening the chat marks nothing.** On a desktop it is open from the start,
  and a phone's sheet used to open on the newest line: either way the player
  was shown the bottom and nothing above it. So `ChatLog` opens on an **Unread
  line** (`.chat-unseen`) above the first line after the mark, with a little of
  what was read above it, and shows the "New messages" button while there is
  more below. The line is placed once, as the log opens, and stays while the
  player reads down from it; nothing missed, no line, and the log opens on the
  newest as before.
- **Seen is what was scrolled into view.** On every scroll, resize and new
  line, `ChatLog` finds the newest line whose whole height has been on screen
  (`data-at` on each line) and reports its time through `onSeen`. A line that
  lands while the log is pinned to the bottom is on screen at once, so it is
  seen at once. "New messages" goes to the newest line, and so marks
  everything. Nothing is reported while the page is hidden (`watching`, from
  `usePageVisible`): a tab left open overnight would mark everything seen.
- **The mark is the time of the newest line shown, never "now"**, which would
  count as seen a line that lands while the call is in flight.
- **`useSeenMark` sends it at most every `SEEN_DELAY_MS` (2s)**, so a fast
  scroll is one call, and at once on `pagehide`, over a keepalive client.
- **It is read and kept as the identity token** (`identity()` for the history,
  `heldIdentity()` for the mark; see [Sessions](sessions.md)): a click token lives
  an hour, so on the next day's visit the identity the cookie resumes, with no
  Turnstile check, is what says who is asking. The mark sends the token in hand
  and never resumes or mints one, since it also goes out as the page closes. A
  refusal for want of an account is silent: a page with no session has nobody
  to keep a mark for.
- In fake mode the mark starts 2.5 minutes back, so two of the opening lines
  count as missed.
- jsdom lays nothing out, so every line counts as on screen in a test. The
  tests of the Unread line stub the layout: lines 50px tall in a log 150px tall.

**The log is never yanked down under someone who scrolled up to read.** It
auto-scrolls only while it is pinned to the bottom (`PINNED_SLACK_PX`);
otherwise a "New messages" pill appears and scrolling back down, by the pill or
by hand, dismisses it.

Every one of these animations is dropped or reduced under
`prefers-reduced-motion: reduce`, keeping the colour and losing the movement.
