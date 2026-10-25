# Chat (`internal/chat/`)

Chat is a separate bounded context, not a feature of the tile game: it shares the process, the transport and the country list, and has its own proto package, domain, storage and edge. Nothing under `internal/chat/` imports `internal/planet/`, and the reverse holds too — and since each module's interior sits behind its own `internal/`, neither now can.

**Always on.** There is no `chat.enabled`: the module is built on every boot, and its database block is required.

**Identity: a username, or a guest code, and always an account.** `SendMessage` reads the click token (`chatv1controller.NewSessionInterceptor`, over `cpconnect.NewSessionReaderInterceptor`, on the key `auth` hands over the internal listener). **A sender with no token, a bad one or a token with no account is refused** with `Unauthenticated` (`messages.ErrNoAccount`) before anything is asked. Every browser that passed Turnstile has an account, a guest one until it signs in, so the web client mints for a post as it does for a click. **The name is the player module's**, asked of `player.v1.InternalService/GetAuthor` on every post: the account's username, or `guest_` and the account's guest code (see [Player](player.md)). Nobody types a guest's name, so no guest can pass for a player or for another guest. **A post the player module could not answer for is refused** with `Unavailable` (`messages.ErrAuthorUnavailable`): no answer within a second, or an error. `log_authors`, a decorator around the adapter, logs it at Error. Messages sent before usernames existed got the prefix from migration `20260917200000_guest_prefix_backfill`, and messages sent before guest codes keep the name a guest typed then, until the retention drops them.

**A message is kept as an account, not as a copy of a name.** `chat.messages.account_id` is who sent it; the name, the crown, the color, the streak and the title worn are read from the player module when the message is *shown*, never stored beside it. That is what makes a rename show on everything its player ever said, and a deleted account stop being named at all — a frozen copy could do neither, and the copy outliving the account it named was a small privacy hole of its own.

**The wire says whose a message is**: `ChatMessage.author_account_id`, on the post's answer, on the stream and in the history. It is empty for a message kept before accounts and for an account that was deleted: nobody is there to open a card of, or to report. A client that has to say which player it means (`ReportPlayer`) sends this id, never the name.

- **The history names a whole window in one ask.** `history_query` collects the distinct accounts of the rows it read and asks `player.v1.InternalService/GetAuthors` once, however many messages each of them sent. An account that says fifty things costs one lookup, not fifty. **A history nobody could be named in is refused** rather than shown anonymous.
- **`SendMessage` still asks `GetAuthor` for one**, because what it publishes has to go out named: everyone already watching is shown who is talking without a second read. That call is also the one that gives a guest its code, which is why the read path uses `GetAuthors` instead — it draws no code, so showing the chat never writes.
- **An account that can no longer be named was deleted**, and reads as `history_query.DeletedName` (`[deleted]`). No username can look like it: `SetName` refuses punctuation. What the account said stays, so a thread keeps its shape.
- **A message with no `account_id` is from before this**, and keeps the `name` and `author_admin` its row carries. Those two columns exist for that alone. **TODO** (see `postgres_message_store.row`): once `chat.storage.retention` has passed since this shipped, every remaining row has an account, and the columns and `history_query`'s fallback to them can go.

The client also sends a UUID it persists locally, kept in the log and **trusted for nothing**. **The sender's address is never public.** It is kept in `chat.messages.ip` for moderation. The salted hash of it that used to go beside every name (`author_tag`) is gone from the wire and the table (migration `20260918210000_drop_tag`): it changed whenever a player changed network, and it told anybody which names shared one.

**Abuse controls live at the edge**, in two interceptors — ahead of decoding the message and well ahead of validating it, so a flood of malformed messages costs a sender exactly what a flood of well-formed ones does. `chat.blockedIPs` cuts an address off from every chat RPC; `chat.rateLimiter` throttles `SendMessage` alone (`GetHistory` is one read on join, and limiting it would punish a page load). **Both are `shared/cpconnect`'s**, the same ones the click chain uses — see [Shared interceptors](interceptors.md). The domain then bounds the message in **runes** (280) and the name (24), validates UTF-8, and **strips control characters** — a newline once let a sender forge a line in the old log file, and a NUL is not valid in a postgres `text`.

Refusal reasons are logged, never returned: a sender learns *that* they were refused, not which check tripped. **The stored text is raw — the frontend must escape it.**

The **vendored VPN lists** do not cover chat: `NewVPNBlockInterceptor` wraps `Click` alone. Chat's blocklist is the same `*cpipblock.Blocklist` type, built by `cpipblock.NewDenyList` from config prefixes instead of vendored data — so entries are CIDRs and a `/24` is one line rather than 256. Extending the vendored lists to chat is therefore a wiring change (build the list in `describeModules` and hand it to both modules, the way `shared/cpcountries` already is), not a second list to write.

**Every message is a row in `chat.messages`**, inserted before it is broadcast: `seq` (the order it was accepted in), `id`, `sent_at`, `account_id` (who sent it), `country`, `ip`, `user_agent`, `text`, `reply_to` (the id of the message a reply answers) — plus `name`, `author_admin` and `author_id`, of which the first two are only ever read, for rows written before `account_id` existed, and the third is a UUID the client made up and nothing trusts. **Postgres is the chat's only copy.** There is no cache in front of it, nothing loaded at boot and nothing flushed: chat is low volume (one message per 3s per address), so every write is one statement and every read one query. `send_message_usecase` inserts (5s timeout) and only then publishes to `inprocess_feed`; `GetHistory` reads the newest `chat.storage.historySize` rows within `retention`, straight from the table: the same window `React` checks a message against (`messages.Window`), and a parity test pins the two. A stream can therefore see two messages sent at the same instant in the other order than `seq`; the client sorts by time.

**The fanout is not storage.** `feed/inprocess_feed` keeps nothing: it hands each update to every open stream and closes one too slow to keep up (`chat.storage.subscriberBuffer`, a key kept from before), never skips an update on it. A client reads the history again each time its stream reopens, which covers whatever it was not listening for.

**Each table has its own `Storage` port and a contract suite** (`messages.StorageContractSuite`, `reactions.StorageContractSuite`, behind the `testing` tag). The postgres stores run it against a real postgres; `inmemory_message_storage` and `inmemory_reaction_storage` run it too, and exist **only for tests** — both files carry the `testing` tag, have no persistence port and are never built into the binary. Use case tests use them instead of hand-written fakes.

**The prune** is `prune_usecase` on a `Runner`, as auth's guest prune is: it deletes messages, then reactions, then announcements, older than `retention`, then the mutes that ended before it and the removals made before it, once at boot and every `pruneInterval`, and `log_prune` logs it. The runner sits inside `cppg.CloseAfter`, so the pool closes after it stops.

The table holds **personal data** — IPs next to user-authored text — so the retention window is a policy decision rather than a cache size.

## Announcements

**The chat also says things on its own**: a line between the messages with no sender, which the client draws
without a bubble. Five kinds today: `bomb`, every bomb that went off, on land or in the sea, `mute`, every
mute an operator gave (see [Mutes](#mutes)), `round`, every round of a season that closed, `fortify`, every
landmass of 50 tiles or more a flag fortified (`{country, ground, landmass, tiles}`: the client names the landmass
from its own table, `ground` for a country's main one; see [Fortify](fortify.md)), and `champion`, the country that
won a season.

- **A separate type and a separate table, not a message with no author.** An announcement has no name, tag, IP,
  text or reactions, and a message has no kind or payload; sharing a base would make every column of one
  optional in the other. So `announcements.Announcement` is `{ID, Kind, At, Payload}`, in `chat.announcements`
  (`id uuid` primary key, `kind` of the type `announcement_kind`, `payload jsonb`, `announced_at`), with its own store, its own contract suite and
  its own case on the wire (`chat.v1.Announcement`: `kind` and `payload` as a JSON string).
- **The id is the fact's**, `announcements.AnnouncementID` (`type AnnouncementID uuid.UUID`): a SHA-1 UUID of what
  the line tells (`https://clickplanet.lol/ledger/<position>/bomb`, `…/fortify`, `mutes/<mute id>`,
  `seasons/<n>/rounds/<n>`, `seasons/<n>/champion`), built by each value's `Announcement(at)`. So a fact told twice
  is one row: `announce_usecase` keeps it with `Storage.AppendOnce` (`ON CONFLICT DO NOTHING`, saying whether it did)
  and publishes it only the first time. That is what makes an event the bus delivers again harmless here. A message's
  id is `text` with a `seq` beside it because it is not trusted to be unique; nothing here has that history. A read
  orders by `announced_at`, then `id`.
- **The payload is the template's values, not the sentence.** The client writes the line, so a new wording, or
  a translation, needs no migration. Each kind owns its payload's shape: `announcements.Bomb` is
  `{country, ground, tile, cleared}`, `ground` and `tile` absent in the sea. **A client shows nothing for a kind
  it does not know**, the way it skips an unknown `oneof` case, so a new kind ships server first. On the server
  it is a value added to `announcement_kind` in a migration, a constant in `announcements.Kinds` and a name in
  `history_query`'s `kinds`; the parity test writes every one of `announcements.Kinds` through the store and reads
  it back.
- **How a bomb gets here**: the ledger relay tells `planet.v1.BombLanded` once the ledger flushed the bomb, with its
  position and the ground read off `clicks.Borders`; `bomb_landed_subscriber` turns it into a `Bomb` and
  `announce_usecase` keeps it, then publishes it on `inprocess_feed`. The announcement's time is the event's
  `landed_at`, to the second the ledger keeps. A bomb Redis did not take is a line missed, never a bomb.
- **How a round gets here**: `seasons` keeps `seasons.v1.RoundClosed` in its outbox with the round's results;
  `round_closed_subscriber` turns it into a `Round`, payload `{number, finale, podium}`, and `announce_usecase`
  keeps it. The podium is `announcements.RoundOf`'s rule: the places ranked 3rd or better, ties included. The
  announcement's time is the round's end, not the minute it closed.
- **How a champion gets here, once**: `seasons` keeps `seasons.v1.SeasonClosed` in its outbox right after the
  finale's `RoundClosed`; `season_closed_subscriber` keeps the countries ranked 1st (`announcements.ChampionOf`;
  level on everything, they win together) and `announce_usecase` keeps a `{season, countries}` payload at the
  season's end. A season nobody scored in announces nothing. Its id is the season's, so a second `SeasonClosed`,
  from a relay that sent it again or a retried close, shows nothing new.
- **`GetHistory` returns them beside the messages**, in `announcements`, the newest `historySize` within
  `retention`, bounded apart from the messages so a burst of bombs never pushes one out. The client puts the two
  lists in one by time. **Once the messages fill the window, none is older than the oldest of them**
  (the `beginning` CTE in `history_query`): the two caps are apart, so 200 bombs reached days past 200 messages, and all of
  them sat in a pile on top of the chat.
- **Not personal data**, but the prune deletes them past `retention` with the messages they sit between.

## Mutes

**An operator can take a player's chat away and leave it the map**: `chat.v1.AdminService/Mute(account_id,
duration)`, on the admin listener (see [Operator tools](admin.md)). A muted caller's
`SendMessage` and `React` are refused; it still reads the chat and plays.

- **A mute holds the account and its network.** The network is the address of the account's latest message
  (`messages.Storage.LatestAddress`, by `seq`), as `mutes.ScopeOf` reads it: the /64 over IPv6, the address over
  IPv4, the throttle's unit (`cpipscope`). Never wider: Free Mobile's subscribers share a /32. So a new guest in a
  private tab on the same line is muted from its first post. An account that never posted has no network, and
  only it is muted. **An IPv4 address can be a carrier's NAT**: a mute on one holds the strangers behind it too.
- **The rule is `Mute.ApplicableTo`**: a caller is muted while a mute names its account or its network and has not
  ended. `postgres_mute_store` asks the same in SQL (a mute with no network is `NULL`, which matches nobody), and
  the contract suite runs both stores.
- **Two decorators refuse, before anything else**: `muting_send_message` and `muting_react` ask
  `mutes.Book.MuteError` with the account and the request's address. A refused post is never named, kept nor
  published, and a refused reaction never written. The use cases did not change.
- **The refusal is `PermissionDenied` with a `chat.v1.MuteRefusal` detail** (`chatmute.Refusal`): the end of the
  mute that ends last. The blocklist's denial carries no detail, which is how the client tells the two apart; an
  older client shows its "not allowed to post" line.
- **No duration is one hour** (`mutes.DefaultDuration`). Otherwise whole seconds, and a negative or fractional one
  is `InvalidArgument`, as is an id that is not an account. A second mute is a second row, so it never shortens one
  that ends later.
- **Every mute is announced, after it is kept, by an event.** `mute_usecase` only keeps the mute, with an id from
  its `IDProvider` (`uuid_id_provider`; tests use `mutes.SequentialIDs`). `publishing_mute` then publishes
  `chat.v1.AccountMuted` with that id, and the chat hears it itself (`chat-announcements-mutes`):
  `account_muted_subscriber` hands it to `announce_mute_usecase`, which names the account (`GetAuthor`) and gives an
  `announcements.Muted`, payload `{name, seconds}`, named after the mute, to `announce_usecase`. So the operator's
  call never fails on the announcement, and a mute that was not kept is never announced. The client writes "<name>
  has been muted for one hour", under the name the account had when the event was heard. A mute Redis did not take
  is a line missed, never the mute; a failure to name the account is tried again, logged by `log_subscriber`.
- **`audit_mute` logs every call at Warn** (`admin chat mute`), with the network it holds.
- **Kept in `chat.mutes`** (`id`, `account_id`, `scope`, `muted_at`, `muted_until`). The network is personal data,
  so the prune deletes a mute that ended more than `retention` ago. There is no unmute RPC: deleting the rows ends
  a mute early (`deploy/vps/README.md`).
- `e2e/mute_test.go` mutes over the console listener and sees the player, its reaction and a fresh guest on its
  network refused, and the line reach the history.

## Removals

**An operator can take a message out of the chat**: `chat.v1.AdminService/RemoveMessage(message_id)`, on the admin
listener, for a report it acts on. The notice-and-action answer to a message, beside the mute for its sender.

- **The row stays, as evidence**: a removal is a row of its own in `chat.removals` (`message_id`, `removed_at`;
  migration `20261018120000_removals`), never a delete or a flag on the message. The prune deletes it with the
  messages, past `retention`.
- **It keeps its place and its author, and loses its text and its reactions**: the history answers it with
  `removed` and no text, and the quote of every reply to it too (`history_query`). The text never leaves the server
  again. The client writes "This message has been deleted".
- **Nobody acts on it**: `Store.Shown` and `Store.Message` count it in the window and refuse it, so a reaction or a
  reply to it is `NotFound`. The parity test runs removals through the history, `React` and `SendMessage`.
- **Only a kept message can be removed** (`Store.Kept`: within the retention, removed or not), else
  `removals.ErrUnknownMessage`, `NotFound`. Removing twice keeps the latest time.
- **Every open stream is told**: `remove_message_usecase` publishes `feed.MessageRemoved` once the removal is kept,
  and the stream sends `message_removed`, which the client empties at once. A client that missed it shows the
  message until its next load.
- **`chat.v1.AdminService/GetMessage(message_id)`** is the message as it was sent, removed or not, with
  `removed_at`, for the operator who reads a report (`message_query`). `NotFound` past the retention.
- **`chat.v1.AdminService/ListMessages(before_message_id | after_message_id, limit, account_id)`** is the chat as it was sent,
  removed or not, a page at a time (`messages_query`). **A message's id is a cursor**: the page just before it, or
  just after it, by `seq` as the history orders them, never with it. Neither is the newest page; both is
  `InvalidArgument` (`ErrTwoCursors`), and a cursor past the retention is `NotFound`. The report page reads one page
  on each side of the reported message, and the next page from the oldest or the newest message it holds.
  - **With an `account_id`, it is that account's messages alone**, cursors included: the page of its messages just
    before or just after the cursor, which may be anybody's message. A report from a player's card names no
    message, so the report page lists the reported account's this way, and reads the whole chat around any one of
    them. An id that is not a UUID is `InvalidArgument` (`ErrInvalidAccount`): read as everyone, it would show other
    people's messages as the account's. The two statements of one author ride the `(account_id)` index.
  - **A page is oldest first**, 50 messages unless asked, 200 at most. Fewer than the limit is the end: nothing more
    kept before it, or nothing more sent after it. Before the cursor it stops at the retention.
  - **Each is a `KeptMessage`**: its text as sent and its `removed_at`, as `GetMessage` answers, its country, its
    time, and what it answers as a `KeptQuote` (its id, its author's name, its text as sent and its `removed_at`;
    unset past the retention, as the history's quote is).
  - **Each is named as the chat names it now** (`author_name`): one `GetAuthors` for every account the page names,
    quotes included; the name it was sent under for a message kept before accounts; `messages_query.DeletedName`
    for an account nobody can name, whose id is still answered, since the operator tools take it. A failure to name
    refuses the read, as the history's does.
  - **It is two statements**: the cursor's `seq` (`NotFound` when it is not kept), then the page, one of two windows
    (`older`, `newer`) under one projection that joins the removals and the quote. The newest page is `older` from
    the top. Each window walks the primary key from the cursor.
- **`audit_remove` logs every call at Warn** (`admin chat message removal`).
- `e2e/report_actions_test.go` removes a reported message over the console listener, and sees it stay in the
  history with no text, its quote with none, a reaction refused, and the text still read for the operator; and
  reads a page on each side of a reported reply: the message it answers before it, the answer to it after, each
  named as the chat named it.

## Replies

**A message may answer another one**: `SendMessageRequest.reply_to_message_id`. The reply is an ordinary message in
the one thread. What marks it is `ChatMessage.reply_to`, a `Quote` of the message it answers (its id, its author's
name and color, its text), which the client draws above the reply.

- **A reply keeps the id of the message it answers and nothing else**: `chat.messages.reply_to` (migration
  `20261011120000_replies`, with an index on `id` for the read). What that message said and who said it are read
  when the reply is shown, for the reason a name is: a rename shows on every quote, and a deleted account reads
  `[deleted]` there too.
- **One level.** A quote is the message answered, never what that one answered: a reply to a reply quotes the
  reply alone.
- **Only a message in the window can be answered** (`messages.ErrNotShown` → `NotFound`), the rule `React` has.
  `Storage.Message` asks the same question as `Shown` and answers the message, restored. The contract suite pins
  the two against each other, and a parity test against the history.
- **`messages.Thread` makes the reply**: it holds the storage, the authors, the clock and the window, and the use
  case calls one method. `Reply(ctx, draft, id)` answers the draft as it is for no id. Otherwise it reads the
  message answered, names its account with `GetAuthors` (a pure read: answering somebody draws no guest code), and
  answers `draft.Answering(answered)`. An account nobody can name is `messages.DeletedName`; a message from before
  accounts keeps the name its row carries. A message nobody could be asked about refuses the post
  (`ErrAuthorUnavailable`).
- **`Message.Answering(answered)` is the whole rule on the message**: it takes the id, the author and the text of
  the other message as its `Quote`. Nothing outside the package reads a field to build one.
- **The answer and the stream carry the quote**, so everyone watching sees what is answered with no second read.
- **The history reads it in the same statement** (`history_query`: a lateral join on `reply_to`, bounded by the
  retention and not by the window, so a reply still quotes a message that newer ones pushed out). The quoted
  accounts go in the one `GetAuthors` ask. **A message that is gone is not quoted**: once it is pruned, or past the
  retention, its reply reads as a plain message.
- **The server sends the whole text.** How much of it is shown is the client's (100 characters).
- `e2e/chat_test.go` replies over HTTP and sees the quote follow a rename.

## Reactions

**A message carries reactions from a fixed set**, `chat.v1.Reaction`. The proto enum is the whole list: the frontend draws each one from its own images (see its [Live chat](../../frontend/docs/chat.md#reactions)), and `chatmessage.Reaction` refuses a number the proto does not name with `InvalidArgument`. **The number is what is stored**, so a value is never renumbered or reused.

- **Who reacts** (`reactions.ReactorOf`): the account, `account:<uuid>`, whether it chose a username or not. No account is refused with `Unauthenticated` (`messages.ErrNoAccount`). So two accounts behind one address are two reactors. A reactor gives each reaction at most once per message. Rows from before, `guest:<tag>`, match no caller and age out with the retention.
- **`React` is on or off, not a toggle.** Asking for what is already there changes nothing, is not written and is not published, so a retry cannot flip it twice. It answers the message's counts with `mine` set for the caller.
- **Only a message in the window can be reacted to** (`ErrUnknownMessage` → `NotFound`): nobody is shown any other. `postgres_message_store.Shown` asks the same question as the history, for one id.
- **Saved, then read back, then published, with no lock.** Two reactions at once can publish their tallies in either order, so **each tally carries a version**: `chat.reaction_versions` holds one counter per message, bumped in the same statement as the change (a data-modifying CTE: no row changed, no bump), and read in the same statement as the reactions, so a tally and its version are one snapshot. The client keeps the highest version it has seen per message and drops a lower one. This holds across processes, which an in-memory lock would not. `reactions.Reactions` is a value: `With`/`Without` answer a copy.
- **The stream sends all of a message's counts, not the difference** (`ReactionsChanged`), with their version, so a client that missed a frame is right on the next, and one that gets two out of order keeps the newer. It cannot know who reads it, so `mine` is always false there; the client keeps its own between calls.
- **`GetHistory` marks the caller's own.** It reads the optional token (the session reader covers it): the account's reactions are marked, and a caller with no token has none.
- **A count says who gave it, not only how many** — as accounts, named when it is shown. `Count` carries the same two halves a `messages.Message` does: `Reactors` is what the store holds, and `Names` is who those accounts are to a reader, filled by `reactions.Named` on the way out. So a rename shows under every reaction its player ever gave, for the same reason it shows on every message.
  - **`reactions.AccountOf`** is `ReactorOf` backwards: `account:<uuid>` to an account, and `false` for a `guest:<tag>` row from before guests had accounts, which is nobody and can be named nothing.
  - **The history names everyone in one ask.** `history_query` gathers the senders *and* the people under their reactions, which its SQL reads as accounts, and calls `GetAuthors` once for the lot.
  - **`React` asks once too**, after the write, and uses that one answer for both the caller's response and the `ReactionsChanged` frame — a reader of the stream has no way to ask for itself. A change that changes nothing pays the same ask, since it answers the same list.
  - **Somebody nobody can name is counted without being named**: a deleted account, or one of those guest tags. `Count` still says how many gave it, and a list of reactions is not the place to announce that somebody is gone — which is why this does not use `history_query.DeletedName`.
  - The edge sends the first 20 names per count (`chatmessage.NamedReactors` for a reaction and the stream, `history_query.NamedReactors` for the history) and lets `count` say the rest, so a message everybody piles onto does not carry a name per reader per message.
- **Stored in `chat.reactions`**, their own table and their own store, one row per `(message_id, reaction, reactor)`, with `reacted_at`. **The reactor is already an account**, so there was never anything to migrate here: the table has held the right thing all along. A read replays the rows oldest first, so each reaction keeps the place it first appeared in, and so do the people under it. The prune deletes rows older than `chat.storage.retention`, like messages: the reactor is an account, so it is personal data. It deletes a version whose last change is that old too, which is only ever one whose message's reactions are all gone.
- **Its own rate bucket**, `chat.reactionLimiter` (defaults: 1 a second, 10 in hand), and the blocklist covers `React` too.

## Seen

**The chat keeps, per account, until when it was seen**, so a player who comes back is shown what was said
while it was away: the badge counts it, and the lines light up as the chat opens.

- **A time, not a message id.** `chat.seen` is one row per account, `seen_until`. Messages and announcements are
  two tables with two kinds of id, and a message's id is the client's text and not unique; one time covers both,
  and still means something once the prune deleted the line it was taken from.
- **The client sends the time of the newest line it showed**, never "now": a message landing during the call
  would count as seen. `seen.Until` cuts a time ahead of the server's clock to now, and refuses no time at all
  (`ErrNoTime` → `InvalidArgument`).
- **It only moves forward**: the upsert keeps the `GREATEST` of the two, so two tabs, or a late request, never
  move it back. **The store only writes**: nothing on the write side reads a mark back, so it has no read and no
  contract suite, and `history_query`'s tests seed through it to pin what it keeps.
- **`GetHistory` answers it** (`seen_until_unix_ms`, 0 for none): `history_query` reads `chat.seen` beside the
  window, in parallel with the messages and the announcements. **`MarkSeen` writes it**, through the `seen` store. No account is `Unauthenticated`.
  It has its own rate bucket, `chat.seenLimiter` (1 a second, 10 in hand), and the blocklist covers it.
- **Both take the identity token, not only the click token** (`cpconnect.Identified` in the chat's session reader;
  see [The click token and the identity token](sessions.md#the-click-token-and-the-identity-token)). A click token lives an
  hour, so a player back the next day holds none at load: exactly the visit the mark is for. The identity token
  the cookie resumes names it with no Turnstile check. A seen mark is a write, but only of the caller's own state,
  so it needs no proof of a check.
- **Forgotten with the account**: `account_deleted_subscriber` hears `auth.v1.AccountDeleted` and deletes the row.
  There is no prune: a row lives as long as its account, and auth's guest prune deletes the guests.
- `TestTheIdentityTokenKeepsWhenThePlayerLastSawTheChat` pins the identity path over HTTP, and
  `TestADeletedAccountLosesItsSeenMark` the deletion.

**Chat has its own stream**, `ChatService.ListenForEvents` — see [The live streams](live-streams.md). It replaced a `/ws/chat` websocket that had to be kept apart from the tile one because frames carried a bare protobuf message with no type tag: a second payload on either socket would have been indistinguishable from the first. The `oneof` envelope is exactly what removes that constraint.

**Sending is an RPC, not a read on the socket**: both publishers lean on `CloseRead` for instant disconnect detection, and the RPC path already has the middleware stack and the interceptors.

`GetHistory` is marked `NO_SIDE_EFFECTS`, so Connect sends it as a GET — but it answers `Cache-Control: no-store`, the opposite of `GetMap`. A client fetches it once on join to seed what the stream then keeps up to date, so a cached answer would show a joiner a chat missing the last few minutes.
