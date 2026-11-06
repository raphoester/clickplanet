# Chat (`internal/chat/`)

Chat is a separate bounded context, not a feature of the tile game: it shares the process, the transport and the country list, and has its own proto package, domain, storage and edge. Nothing under `internal/chat/` imports `internal/planet/`, and the reverse holds too — and since each module's interior sits behind its own `internal/`, neither now can.

**Always on.** There is no `chat.enabled`: the module is built on every boot, and its database block is required.

**Identity: a username, or a guest code, and always an account.** `SendMessage` reads the click token (`chatv1controller.NewSessionInterceptor`, over `cpconnect.NewSessionReaderInterceptor`, with the process's one verifier, `props.Sessions`, on the key `auth` hands over the internal listener). **A sender with no token, a bad one or a token with no account is refused** with `Unauthenticated` (`messages.ErrNoAccount`) before anything is asked. Every browser that passed Turnstile has an account, a guest one until it signs in, so the web client mints for a post as it does for a click. **Every sender is introduced to the player module**, with `player.v1.InternalService/IntroducePlayer` on every post: it gives a guest its code the first time (see [Player](player.md)), and says whether the sender is a guest, which posts no GIF (`Message.CheckSender`). The name stays the player module's: nobody types a guest's name, so no guest can pass for a player or for another guest. **A post the player module could not answer for is refused** with `Unavailable` (`messages.ErrAuthorUnavailable`): no answer within a second, or an error. `log_authors`, a decorator around the adapter, logs it at Error.

**A message is kept as an account, not as a copy of a name.** `chat.messages.account_id` is who sent it; the name, the crown and the color are the player module's, which the page reads by that account when it shows the message (`player.v1.PlayerService/GetNameplates`, see the frontend's [Nameplates](../../frontend/docs/nameplates.md)). That is what makes a rename show on everything its player ever said, and a deleted account stop being named at all — a frozen copy could do neither, and the copy outliving the account it named was a small privacy hole of its own.

**The wire says whose a message is, and nothing else of who**: `ChatMessage.author_account_id`, on the post's answer, on the stream and in the history; `Quote.author_account_id` for what a reply answers; `ReactionCount.reactor_account_ids` for who reacted. A client that has to say which player it means (`ReportPlayer`) sends this id, never the name.

- **Nothing the chat shows asks the player module** but the sender's introduction: the history, a reply and a reaction read accounts from the chat's own tables. Until 2026-11-06 the chat asked `GetPlayers` at every history, reply and reaction, only to copy the names, the crowns and the colors into its answers (`author_name`, `author_admin`, `author_color` and the quote's, and the reactors' names, all `reserved` now).
- **An account the other players can no longer name was deleted or asked deleted**: the chat still answers its id, and the page draws it `[deleted]`, with no card and no report, since `GetNameplates` leaves it out. No username can look like it: `SetName` refuses punctuation. What the account said stays, so a thread keeps its shape.
- **The copies of a name are gone from the table too**: `chat.messages.name` and `author_admin` (migration `20261106120000_drop_author_copies`). They were read only for rows from before accounts (2026-09-21), and the 30-day retention dropped the last of those; `SendMessage` had been writing them empty. A row with no `account_id` would now be no one's: `[deleted]` on the page and in the operator's list.

The client also sends a UUID it persists locally, kept in the log and **trusted for nothing**. **The sender's address is never public.** It is kept in `chat.messages.ip` for moderation. The salted hash of it that used to go beside every name (`author_tag`) is gone from the wire and the table (migration `20260918210000_drop_tag`): it changed whenever a player changed network, and it told anybody which names shared one.

**Abuse controls live at the edge**, in two interceptors — ahead of decoding the message and well ahead of validating it, so a flood of malformed messages costs a sender exactly what a flood of well-formed ones does. `chat.blockedIPs` cuts an address off from every chat RPC; `chat.rateLimiter` throttles `SendMessage` alone (`GetHistory` is one read on join, and limiting it would punish a page load). **Both are `shared/cpconnect`'s**, the same ones the click chain uses — see [Shared interceptors](interceptors.md). The domain then bounds the message in **runes** (280) and the name (24), validates UTF-8, and **strips control characters** — a newline once let a sender forge a line in the old log file, and a NUL is not valid in a postgres `text`. **A text with a character of a historic script is refused** (`cpscripts.Historic`, see [Shared](shared.md)): lines of wide cuneiform signs were spammed in Nov 2026. Every alphabet written today, emojis, punctuation and symbols pass.

Refusal reasons are logged, never returned: a sender learns *that* they were refused, not which check tripped. **The stored text is raw — the frontend must escape it.**

The **vendored VPN lists** do not cover chat: `NewVPNBlockInterceptor` wraps `Click` alone. Chat's blocklist is the same `*cpipblock.Blocklist` type, built by `cpipblock.NewDenyList` from config prefixes instead of vendored data — so entries are CIDRs and a `/24` is one line rather than 256. Extending the vendored lists to chat is therefore a wiring change (build the list in `describeModules` and hand it to both modules, the way `shared/cpcountries` already is), not a second list to write.

**Every message is a row in `chat.messages`**, inserted before it is broadcast: `seq` (the order it was accepted in), `id`, `sent_at`, `account_id` (who sent it), `room` (see [Rooms](#rooms)), `country`, `ip`, `user_agent`, `text`, `reply_to` (the id of the message a reply answers) — plus `author_id`, a UUID the client made up and nothing trusts. **Postgres is the chat's only copy.** There is no cache in front of it, nothing loaded at boot and nothing flushed: chat is low volume (one message per 3s per address), so every write is one statement and every read one query. `send_message_usecase` inserts (5s timeout) and only then publishes to `inprocess_feed`; `GetHistory` reads the newest `chat.storage.historySize` rows within `retention`, straight from the table: the same window `React` checks a message against (`messages.Window`), and a parity test pins the two. A stream can therefore see two messages sent at the same instant in the other order than `seq`; the client sorts by time.

**The fanout is not storage.** `feed/inprocess_feed` keeps nothing: it hands each update to every open stream and closes one too slow to keep up (`chat.storage.subscriberBuffer`, a key kept from before), never skips an update on it. A client reads the history again each time its stream reopens, which covers whatever it was not listening for.

**Each table has its own `Storage` port and a contract suite** (`messages.StorageContractSuite`, `reactions.StorageContractSuite`, behind the `testing` tag). The postgres stores run it against a real postgres; `inmemory_message_storage` and `inmemory_reaction_storage` run it too, and exist **only for tests** — both files carry the `testing` tag, have no persistence port and are never built into the binary. Use case tests use them instead of hand-written fakes.

**The prune** is `prune_usecase` on a `Runner`, as auth's guest prune is: it deletes messages, then reactions, then announcements, older than `retention`, then the mutes that ended before it and the removals made before it, then the moderation history older than its own `moderationRetention` (see [The moderation history](#the-moderation-history)), once at boot and every `pruneInterval`, and `log_prune` logs it. Each set of tables is `prune_usecase.KeptFor(retention, pruners...)`. The runner sits inside `cppg.CloseAfter`, so the pool closes after it stops.

The table holds **personal data** — IPs next to user-authored text — so the retention window is a policy decision rather than a cache size.

## Announcements

**The chat also says things on its own**: a line between the messages with no sender, which the client draws
without a bubble. Six kinds today: `bomb`, every bomb that went off, on land or in the sea, `mute`, every
mute an operator gave (see [Mutes](#mutes)), `round`, every day and battle of a season scoring scored, `fortify`, every
landmass of 50 tiles or more a flag fortified (`{country, ground, landmass, tiles}`: the client names the landmass
from its own table, `ground` for a country's main one; see [Fortify](fortify.md)), `champion`, the country that
won a season, and `flag_mission`, every flag that finished its mission of the day (see
[The flag's mission](missions.md#the-flags-mission)), told in that flag's room too.

- **A separate type and a separate table, not a message with no author.** An announcement has no name, tag, IP,
  text or reactions, and a message has no kind or payload; sharing a base would make every column of one
  optional in the other. So `announcements.Announcement` is `{ID, Kind, At, Room, Payload}`, in `chat.announcements`
  (`id uuid` primary key, `kind` of the type `announcement_kind`, `room`, `payload jsonb`, `announced_at`), with its own store, its own contract suite and
  its own case on the wire (`chat.v1.Announcement`: `kind` and `payload` as a JSON string).
- **The id is the fact's**, `announcements.AnnouncementID` (`type AnnouncementID uuid.UUID`): a SHA-1 UUID of what
  the line tells (`https://clickplanet.lol/ledger/<position>/bomb`, `…/fortify`, `mutes/<mute id>`,
  `seasons/<n>/rounds/<n>`, `seasons/<n>/champion`, `flag-missions/<mission id>`), built by each value's `Announcement(at)`. So a fact told twice
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
  position and the ground and the landmass read off `clicks.Borders`. The landmass is what the line names: Tasmania's
  ground is `au`, and the line says "Tasmania". `bomb_landed_subscriber` turns it into a `Bomb` and
  `announce_usecase` keeps it, then publishes it on `inprocess_feed`. The announcement's time is the event's
  `landed_at`, to the second the ledger keeps. A bomb Redis did not take is a line missed, never a bomb.
- **How a round gets here**: scoring scores a round and tells `scoring.v1.RoundScored`; the gamemaster hears it and
  tells, in the game's words, `gamemaster.v1.DayEnded` or `BattleEnded`, with the round's id and no numbers.
  `day_ended_subscriber` (group `chat-announcements-days`) and `battle_ended_subscriber`
  (`chat-announcements-battles`) name the round, `announcements.DayOf(season, day)` or
  `announcements.BattleOf(season, number, ends_season, times)`, and `announce_round_usecase` reads its results by
  id from `scoring.v1.InternalService/GetRound` on the internal listener (`rpc_scoring_results`, one second), places
  them (`Round.Placed`: the places ranked 3rd or better that scored, ties included) and hands the line to
  `announce_usecase`. Payload `{number, podium}`; **a battle** adds `own: true`, its `times`, and `finale: true` when
  it ran to the season's end, and is told under an id of its own (`seasons/<season>/own-rounds/<number>`), since its
  number counts the season's battles and not the days. The announcement's time is the event's `ended_at`. A round
  scoring cannot answer for is refused and logged, so the bus delivers it again.
- **How a champion gets here, once**: scoring closes the season's book and tells `scoring.v1.BookClosed`; the
  gamemaster tells `gamemaster.v1.SeasonEnded` with the book's id. `season_ended_subscriber`
  (`chat-announcements-champions`) and `announce_champion_usecase` read the book's table from
  `InternalService/GetTable` (refused while the book is not closed) and keep the countries ranked 1st
  (`announcements.ChampionOf`; level on everything, they win together) as a `{season, countries}` payload at the
  season's end. A season nobody scored in announces nothing. Its id is the season's, so a second `SeasonEnded`
  shows nothing new.
- **How a flag mission gets here**: missions finishes a flag's mission and tells `missions.v1.FlagMissionDone`, from its
  outbox, in the transaction that gives the flag its points: the flag, the mission as its players see it
  (`missions.v1.FlagMission`) and the time of the take or the shield broken that finished it.
  `flag_mission_done_subscriber` (`chat-announcements-flag-missions`) makes it an `announcements.FlagMission`, under the
  mission's id (`flag-missions/<mission id>`), and `announce_usecase` keeps it. Payload `{country, points, mission,
  ground | rival, goal}`: `mission` is `take_in`, with the `ground` to take tiles in, or `break_shields`, with the
  `rival` whose shields to break. **A kind the chat does not know is still told**, as `{country, points}` alone, so a new
  kind of flag mission in missions never silences the line. It is the only kind told in a flag's room: see
  [Rooms](#rooms).
- **`GetHistory` returns them beside the messages**, in `announcements`, the newest `historySize` within
  `retention`, bounded apart from the messages so a burst of bombs never pushes one out. The client puts the two
  lists in one by time. **Once the messages fill the window, none is older than the oldest of them**
  (the `beginning` CTE in `history_query`): the two caps are apart, so 200 bombs reached days past 200 messages, and all of
  them sat in a pile on top of the chat.
- **Not personal data**, but the prune deletes them past `retention` with the messages they sit between.

## Mutes

**An operator can take a player's chat away and leave it the map**: `chat.v1.AdminService/Mute(account_id,
duration, kind, reason)`, on the admin listener (see [Operator tools](admin.md)). A muted caller's
`SendMessage` and `React` are refused; it still reads the chat and plays.

- **A mute has a kind: what it takes away** (`mutes.Kind`, `chat.v1.MuteKind`). `Chat`, the zero value on the wire
  and in Go, is the whole chat, every post and every reaction: what a mute was before kinds. `Gifs` is a post with a
  GIF alone: the player still writes and reacts. **One concept, not two**: both kinds are the same `Mute`, kept in the
  same table, held on the account and its network by the same rule, refused by the same `Book`, given, lifted and
  read by the same three calls, each of which names its kind. Each kind holds apart, and neither ends the other. A
  kind the proto does not name is `InvalidArgument`, never the whole chat by default.
- **`muting_send_message` asks the whole chat first, then the GIFs for a post with one**, so a player muted both ways
  is told the mute that stops it from writing at all. `muting_react` asks the whole chat alone.
- **Every mute is announced with its kind**: `chat.v1.AccountMuted` carries the `kind`, and so does the line's payload,
  so the client writes "<name> may not post GIFs for one day" for a mute of the GIFs. A guest posts no GIF anyway
  (see [GIFs](#gifs)), so a mute of the GIFs on a network matters for a signed-in account on the same line.
- **A mute holds the account and its network.** The network is the address of the account's latest message
  (`messages.Storage.LatestAddress`, by `seq`), as `mutes.ScopeOf` reads it: the /64 over IPv6, the address over
  IPv4, the throttle's unit (`cpipscope`). Never wider: Free Mobile's subscribers share a /32. So a new guest in a
  private tab on the same line is muted from its first post. An account that never posted has no network, and
  only it is muted. **An IPv4 address can be a carrier's NAT**: a mute on one holds the strangers behind it too.
- **The rule is `Mute.ApplicableTo`**: a caller is muted of a kind while a mute of that kind names its account or its
  network and has not ended. `postgres_mute_store` asks the same in SQL (a mute with no network is `NULL`, which matches nobody), and
  the contract suite runs both stores.
- **Two decorators refuse, before anything else**: `muting_send_message` and `muting_react` ask
  `mutes.Book.MuteError` with the account and the request's address. A refused post is never named, kept nor
  published, and a refused reaction never written. The use cases did not change.
- **The refusal is `PermissionDenied` with a `chat.v1.MuteRefusal` detail** (`chatmute.Refusal`): the end of the
  mute that ends last, and its `kind`, so the client says "muted" or "no GIFs". The blocklist's denial carries no
  detail, which is how the client tells the two apart; an older client shows its "not allowed to post" line.
- **No duration is one hour** (`mutes.DefaultDuration`). Otherwise whole seconds, and a negative or fractional one
  is `InvalidArgument`, as is an id that is not an account. A second mute is a second row, so it never shortens one
  that ends later.
- **Every mute is announced, after it is kept, by an event.** `mute_usecase` only keeps the mute, with an id from
  its `IDProvider` (`uuid_id_provider`; tests use `mutes.SequentialIDs`). `publishing_mute` then publishes
  `chat.v1.AccountMuted` with that id, and the chat hears it itself (`chat-announcements-mutes`):
  `account_muted_subscriber` hands it to `announce_mute_usecase`, which names the account (`IntroducePlayer`) and gives an
  `announcements.Muted`, payload `{name, kind, seconds}` (`kind` is `chat` or `gifs`; a line kept before kinds has none,
  and was the whole chat's), named after the mute, to `announce_usecase`. An event whose kind the proto does not name is
  refused and logged. So the operator's
  call never fails on the announcement, and a mute that was not kept is never announced. The client writes "<name>
  has been muted for one hour", or "<name> may not post GIFs for one hour", under the name the account had when the event was heard. A mute Redis did not take
  is a line missed, never the mute; a failure to name the account is tried again, logged by `log_subscriber`.
- **`audit_mute` logs every call at Warn** (`admin chat mute`), with the network it holds and the reason.
- **A mute takes a `reason`**, optional, and is recorded in the [moderation history](#the-moderation-history) in the
  transaction that keeps it: never a mute without its record.
- **Kept in `chat.mutes`** (`id`, `account_id`, `scope`, `kind`, `muted_at`, `muted_until`; `kind` is the type
  `mute_kind`, migration `20261104130000_mute_kinds`, which made every mute before it one of the whole chat). The network is personal data,
  so the prune deletes a mute that ended more than `retention` ago.
- **`Unmute(account_id, kind, reason)` ends now every mute of the kind given to the account**, with the network each holds. `unmute_usecase`
  reads the account's mutes (`MutesOf`), and saves each that has not `Ended` as `Lifted` now, in one transaction: the
  row stays, with `muted_until` moved to now, so the prune still takes it. `Save` is an upsert by id. A mute given
  to another account on the same network still holds it. None running is `NotFound`. The answer is the end the
  mute that would have ended last had. No announcement and no event: a client tries each post again, so the next
  one goes through. `audit_unmute` logs every call at Warn (`admin chat unmute`). It takes a `reason` too, and an
  unmute that lifts a mute is recorded in the history in its transaction; one that lifts none is `NotFound` and no
  action.
- **`GetMute(account_id, kind)` says whether a mute of the kind holds the account now** (`mute_query`): a mute given to it, or to
  another account on the network it last posted from, the one that ends last, with the account it was given to.
  Nothing set when none holds it. It reads the latest address, not the request's, so a player that moved network
  since its last post may read otherwise than its next post is judged.
- `muting_send_message` and `muting_react` refuse the player, its reaction and a fresh guest on its network;
  `announce_mute_usecase` writes the line.

## Removals

**An operator can take a message out of the chat**: `chat.v1.AdminService/RemoveMessage(message_id, reason)`, on the admin
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
  - **Each is named on the server** (`author_name`), since the back office and the connector cannot compose names:
    one `GetPlayers` for every account the page names, quotes included; `messages_query.DeletedName` for an account
    nobody can name, or a row with no account, whose id is still answered when it has one, since the operator tools
    take it. A failure to name refuses the read.
  - **It is two statements**: the cursor's `seq` (`NotFound` when it is not kept), then the page, one of two windows
    (`older`, `newer`) under one projection that joins the removals and the quote. The newest page is `older` from
    the top. Each window walks the primary key from the cursor.
- **`audit_remove` logs every call at Warn** (`admin chat message removal`), with the reason.
- **`chat.v1.AdminService/RemoveMessages(removals)` removes many at once**, each a `MessageRemoval` (`message_id`,
  `reason`): a moderation run's removals in one call, so one approval on claude.ai or in Claude Code instead of one per
  message (`chat_remove_messages`). `remove_messages_usecase` is `remove_message_usecase` for a list, **all or none**:
  - **Checked before anything is read**: 1 to 200 messages (`MaxRemovals`, a page of `ListMessages`; `ErrNoRemoval`,
    `ErrTooManyRemovals`), none twice (`ErrRemovedTwice`: the history keeps a message's first removal, so the second
    reason would be lost without a word), and every reason (`moderation.ReasonOf`). An empty id is refused by the
    handler, as `RemoveMessage`'s is.
  - **Then every message is read** (`KeptMessage`, within the retention), then every removal and its record are kept
    in one transaction, and only then is each stream told (`feed.MessageRemoved`, one per message), all at one time.
  - **One refused refuses them all, and the refusal names it**: `NotFound` for a message past the retention,
    `InvalidArgument` for the rest. The caller sends the others again without it.
  - **A message already removed is removed again** and keeps its first action in the history, as `RemoveMessage`
    does, so a run tried again after a failure removes what was left and counts nothing twice.
  - **`audit_remove_messages` logs one `admin chat message removal` line per message**, with its reason and `batch`
    (how many the call removed), so a search for a message finds it however it was removed; a refused call is one
    `admin chat message removals failed` line with every id.
- **A removal takes a `reason`**, and is recorded in the history in the transaction that keeps it, on the message's
  author, with **a copy of what the message said**: its text, or its GIF's address, never its address, so it outlives
  the chat's retention. `remove_message_usecase` reads the message with `KeptMessage` (removed or not, within the
  retention; `messages.ErrNotKept` past it). A message removed twice is one action, the first: a call tried again never
  counts twice (`moderation_removal_key`).
- `remove_message_usecase` keeps the removal; the history query's suite shows the message in its place with no
  text and its quote with none, the message store refuses a reaction to it, and `message_query` still reads the
  text for the operator. `messages_query` reads a page on each side of a message; player's `players_query` pins
  that `GetPlayers`, and so `GetNameplates`, names each account as `IntroducePlayer` named it in the chat.

## The moderation history

**Every mute, unmute and removal an operator made is kept, per account, with the reason given and who made it**, past
the chat's own retention: `moderation`, `chat.moderation`, read by
`chat.v1.AdminService/ListModeration(account_id, since, limit)`. It is what Claude's runs count their steps from (see
[Chat moderation](../../../docs/moderation.md)), and what the operator audits Claude by. The operator's renames are
player's: see [Who owns which history](#who-owns-which-history).

- **One row per action**: `account_id` (the account it was taken on, never the network a mute also holds; NULL for a
  message kept before accounts), `action` (the type `moderation_action`: `mute`, `unmute`, `removal`), `acted_at`;
  a mute's and an unmute's `mute_kind`, a mute's `seconds`; a removal's `message_id` and its copy, `text` or `gif_url`,
  `gif_width`, `gif_height`; then `reason` and who: `operator_id` and `client` (migration `20261106130000_moderation`,
  `CHECK`s that each action has its own columns and no other's).
- **Written in the action's transaction**: `mute_usecase`, `unmute_usecase`, `remove_message_usecase` and
  `remove_messages_usecase` (one for all its removals) each open one (`InTx`) and call `moderation.Storage.Record` beside their own store, so a mute, an unmute or a removal is never kept
  without its record, nor a record without it. The domain builds it (`moderation.NewMute`, `NewUnmute`, `NewRemoval`);
  the store only writes, and the query reads.
- **The reason** is `moderation.ReasonOf`: support's text rule (`notes.TextOf`: valid UTF-8, lines kept, every other
  control character stripped, the ends trimmed), but empty is no reason, and 500 characters at most. A reason it refuses
  is `InvalidArgument`, before anything is done. An older caller sends none, and its action is kept with none.
- **Who** is what the admin gate admitted: the operator's account and, for an app the operator connected, its name
  (`client`, `Claude`), as the `admin call` line says it. The gate puts it on the call's context
  (`cpctx.Operator`, see [Operator tools](admin.md)) and each handler reads it there (`chatoperator.Of`). A call on the
  console listener has no gate, and is kept by nobody: no operator, no client.
- **The removed text is kept, on purpose** (decided with the owner on 2026-11-06): the reason alone cannot be checked
  against what was said once the message is pruned, and Claude removes on its own. The copy has no address beside it,
  and is personal data all the same, so the privacy page lists it.
- **`ListModeration`** (`moderation_query`): newest first (`acted_at`, then `id`), one account's or every account's (an
  id that is not a UUID is `InvalidArgument`, never read as everyone), taken at `since` or after (unset: all of it),
  50 unless asked, 500 at most. Each is a `ModerationAction`: its id, the account, when, a `oneof` of `MuteAction` (kind,
  duration), `UnmuteAction` (kind) and `RemovalAction` (message id and its copy), the reason, `operator_account_id`
  and `client`. `NO_SIDE_EFFECTS`, so it is a read-only tool of the connector. `no-store`. A kind the wire cannot carry
  refuses the read (`ErrUnknownAction`, `ErrUnknownKind`).
- **Its own retention**, `chat.storage.moderationRetention` (90 days by default, as support keeps a note's sender), by
  the chat's prune.
- **Forgotten with the account**: `chat-moderation-forgotten` hears `player.v1.PlayerForgotten` and
  `forget_moderation_usecase` deletes the actions taken on the account, copies included, and takes it off the actions it
  took as an operator, which stay with nobody.
- The use cases' tests pin each record and that a failed record fails the action; `moderation_query`'s suite seeds
  through the store and reads every kind back; the store's suite pins the one removal per message, the prune and the
  forget.

### Who owns which history

**Each context keeps the history of what it does**: the chat its mutes, unmutes and removals, player its renames
(`player.v1.AdminService/ListRenames`, see [Player](player.md)). The back office and Claude read both.

- **Why not one context hearing the other's events**: a record is written in the transaction of the action it records,
  so there is no action without its record, and a run reads it the moment the call answers. Through an event, player
  would need a `PlayerRenamed` carrying the reason and the operator, an outbox to send it, and chat a subscriber, for a
  log whose only reader is the operator; the run that renamed would read its history before the record arrived.
- **Why not a module of its own**: it would own no rule. Mutes and removals are the chat's, a name is player's, and the
  steps Claude counts are the chat's alone (a mute and a removal); renames go in the digest.

## Replies

**A message may answer another one**: `SendMessageRequest.reply_to_message_id`. The reply is an ordinary message in
the one thread. What marks it is `ChatMessage.reply_to`, a `Quote` of the message it answers (its id, its author's
account, its text), which the client draws above the reply, its author named by the page.

- **A reply keeps the id of the message it answers and nothing else**: `chat.messages.reply_to` (migration
  `20261011120000_replies`, with an index on `id` for the read). What that message said and whose it is are read
  when the reply is shown, and the page names the account, for the reason a message's: a rename shows on every
  quote, and a deleted account reads `[deleted]` there too.
- **One level.** A quote is the message answered, never what that one answered: a reply to a reply quotes the
  reply alone.
- **Only a message in the window can be answered** (`messages.ErrNotShown` → `NotFound`), the rule `React` has.
  `Storage.Message` asks the same question as `Shown` and answers the message, restored. The contract suite pins
  the two against each other, and a parity test against the history.
- **`messages.Thread` makes the reply**: it holds the storage, the clock and the window, and the use case calls one
  method. `Reply(ctx, draft, id)` answers the draft as it is for no id. Otherwise it reads the message answered and
  answers `draft.Answering(answered)`.
- **`Message.Answering(answered)` is the whole rule on the message**: it takes the id, the account and the text of
  the other message as its `Quote`. Nothing outside the package reads a field to build one.
- **The answer and the stream carry the quote**, so everyone watching sees what is answered with no second read.
- **The history reads it in the same statement** (`history_query`: a lateral join on `reply_to`, bounded by the
  retention and not by the window, so a reply still quotes a message that newer ones pushed out). **A message that is
  gone is not quoted**: once it is pruned, or past the retention, its reply reads as a plain message.
- **The server sends the whole text.** How much of it is shown is the client's (100 characters).
- `thread_test.go` and the history query's suite pin that a quote carries its author's account.

## Reactions

**A message carries reactions from a fixed set**, `chat.v1.Reaction`. The proto enum is the whole list: the frontend draws each one from its own images (see its [Live chat](../../frontend/docs/chat.md#reactions)), and `chatmessage.Reaction` refuses a number the proto does not name with `InvalidArgument`. **The number is what is stored**, so a value is never renumbered or reused.

- **Who reacts** (`reactions.ReactorOf`): the account, `account:<uuid>`, whether it chose a username or not. No account is refused with `Unauthenticated` (`messages.ErrNoAccount`). So two accounts behind one address are two reactors. A reactor gives each reaction at most once per message. Rows from before, `guest:<tag>`, match no caller and age out with the retention.
- **`React` is on or off, not a toggle.** Asking for what is already there changes nothing, is not written and is not published, so a retry cannot flip it twice. It answers the message's counts with `mine` set for the caller.
- **Only a message in the window can be reacted to** (`ErrUnknownMessage` → `NotFound`): nobody is shown any other. `postgres_message_store.Shown` asks the same question as the history, for one id.
- **Saved, then read back, then published, with no lock.** Two reactions at once can publish their tallies in either order, so **each tally carries a version**: `chat.reaction_versions` holds one counter per message, bumped in the same statement as the change (a data-modifying CTE: no row changed, no bump), and read in the same statement as the reactions, so a tally and its version are one snapshot. The client keeps the highest version it has seen per message and drops a lower one. This holds across processes, which an in-memory lock would not. `reactions.Reactions` is a value: `With`/`Without` answer a copy.
- **The stream sends all of a message's counts, not the difference** (`ReactionsChanged`), with their version, so a client that missed a frame is right on the next, and one that gets two out of order keeps the newer. It cannot know who reads it, so `mine` is always false there; the client keeps its own between calls.
- **`GetHistory` marks the caller's own.** It reads the optional token (the session reader covers it): the account's reactions are marked, and a caller with no token has none.
- **A count says who gave it, not only how many** — as accounts, which the page names when it shows them (`reactor_account_ids`). So a rename shows under every reaction its player ever gave, for the same reason it shows on every message.
  - **`Count.Accounts`** reads them off the reactors, oldest first, with `reactions.AccountOf`, `ReactorOf` backwards: `account:<uuid>` to an account, and `false` for a `guest:<tag>` row from before guests had accounts, which is nobody and is counted only.
  - **Neither `React` nor the history asks anybody**: the history's SQL reads the reactors as accounts in the same statement, and `React` answers the tally it read back, for the caller and the `ReactionsChanged` frame alike.
  - **Somebody nobody can name is counted, and the page leaves it unnamed**: a deleted account. `count` still says how many gave it, and a list of reactions is not the place to announce that somebody is gone.
  - The edge sends the first 20 accounts per count (`chatmessage.NamedReactors` for a reaction and the stream, `history_query.NamedReactors` for the history) and lets `count` say the rest, so a message everybody piles onto does not carry an id per reader per message.
- **Stored in `chat.reactions`**, their own table and their own store, one row per `(message_id, reaction, reactor)`, with `reacted_at`. **The reactor is already an account**, so there was never anything to migrate here: the table has held the right thing all along. A read replays the rows oldest first, so each reaction keeps the place it first appeared in, and so do the people under it. The prune deletes rows older than `chat.storage.retention`, like messages: the reactor is an account, so it is personal data. It deletes a version whose last change is that old too, which is only ever one whose message's reactions are all gone.
- **Its own rate bucket**, `chat.reactionLimiter` (defaults: 1 a second, 10 in hand), and the blocklist covers `React` too.

## Seen

**The chat keeps, per account, until when it was seen**, so a player who comes back is shown what was said
while it was away: the badge counts it, and the lines light up as the chat opens.

- **A time, not a message id.** `chat.seen` is one row per account and room, `seen_until`. Messages and announcements are
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
- **Forgotten with the account**: `player_forgotten_subscriber` hears `player.v1.PlayerForgotten` (`chat-seen-forgotten`),
  which player tells for every account auth deletes, and deletes the row. There is no prune: a row lives as long as its
  account, and auth's guest prune deletes the guests. Until 2026-11-04 it heard `auth.v1.AccountDeleted`
  (`chat-seen`); why the new group misses no deletion is in
  [Player](player.md#the-deploy-that-moved-the-others-to-players-events).
- `TestAnIdentityTokenNamesTheReaderAndTheSeenMarkOnly` pins the identity path, the history query's
  `TestTheSeenMarkOnlyMovesForward` the mark, and `TestADeletedSeenMarkIsGoneAndOnlyThatOne` the deletion.

## GIFs

**A message shows a text or a GIF, never both**, and the contract says so: `oneof body { text, gif }` on
`SendMessageRequest`, `ChatMessage`, `Quote`, and the operator's `GetMessageResponse`, `KeptMessage` and
`KeptQuote`. `text` kept its field number when it moved into the oneof, so the bytes did not change and an older
client reads and sends a text as before. A removed message has no body at all. The client finds the GIF on KLIPY
itself (see the frontend's [Live chat](../../frontend/docs/chat.md#gifs)); the server only checks and keeps what it
was sent.

- **`messages.Gif` is KLIPY's address and size**, kept as KLIPY gave it: its terms forbid rewriting or caching the
  address. `Limits.Body` refuses a GIF with a text, an address that is not `https` on `static.klipy.com`,
  `static1.klipy.com` or `static2.klipy.com` (no user name, no port), and a side of 0 or over 2000:
  `ErrInvalidMessage`. So no player makes a reader's browser load an address it chose.
- **An operator can take a player's GIFs away and leave it its text**: a mute of the kind `Gifs` (see
  [Mutes](#mutes)).
- **A guest sends no GIF**: `Message.SentBy(author)` refuses it with `ErrGuestGif`, `PermissionDenied`, once the
  author is known and before anything is kept. A guest account is free and a private tab is a new one, so a mute
  does not stop a guest who posts GIFs; the same reason a guest's address is not a link. The player module says who
  is a guest: the `guest` of `IntroducePlayer`'s player, and `rpc_player_authors` answers `messages.GuestOf`.
- **Kept in `chat.gifs`**, one row per message that shows one (`message_seq` → `messages.seq`, `url`, `width`,
  `height`; migration `20261104120000_gifs`): a text has no row, so no column stands empty on it. `Append` writes the
  message and its GIF in one statement (a data-modifying CTE). The row goes with its message (`ON DELETE CASCADE`),
  so the prune needs no step of its own.
- **Every read joins it**: the history, the message a reply or a reaction is checked against (so a reply quotes the
  GIF), and the operator's `GetMessage` and `ListMessages`. **A removed message shows no GIF**, nor its quote, as it
  shows no text; the operator still reads it.
- The store's contract suite keeps and reads one back; the history query's suite pins the quote and the removal.

## Rooms

**Each flag has a room of its own beside the chat everyone reads**: only the accounts that play for that flag read
and write there. A `messages.Room` is `World` (stored `world`) or a flag's room, stored as the flag (`fr`). On the
wire each request says its room in `flag_room`: empty for the world, a country for its room (`messages.RoomOf`).

- **Who plays for which flag is planet's**, asked on each call to a flag's room:
  `planet.v1.InternalService/GetAllegianceOf`, with one second to answer (`rpc_planet_flags`). The chat keeps no copy,
  so a new flag counts at once. A call to the world asks nothing.
- **`messages.Rooms.EntryError` is the rule**: nil for the world; `ErrNoAccount` with no account; `ErrNotInRoom` for an
  account that plays for another flag, or for none. A failure to ask planet is the error net's.
- **Every call to a flag's room checks it**: `Drafts.Draft` before it draws an id (a post), `Showing.Message` once it
  read the message's room (a reaction), `get_history_handler` before it reads, `mark_seen_usecase`, and
  `listen_for_events_usecase` once it has subscribed, so a pledge in between is read or heard. `ErrNotInRoom` is
  `FailedPrecondition`, and the client reads its flag again.
- **Every row of `chat.messages` has its room** (migration `20261103120000_rooms`: the rows from before are the
  world's), with an index on `(room, seq)`. **Each room has its own window**: the newest `historySize` of that room, so
  a busy world never pushes a flag's room out. `Storage.Message` reads a message within the window of its own room.
- **A reply stays in its room**: `Message.Answering` refuses a message of another room as `ErrNotShown`.
- **An announcement is told in a room, as a message is, and the world shows every one.** `chat.announcements.room` is
  `world` for bombs, rounds, mutes, fortifies and champions (migration `20261105200000_flag_mission_announcements`:
  every row from before is the world's), and the flag's room for a flag mission (`Announcement.WithRoom`). A flag's
  room shows what was told in it alone (`Announcement.In`): France's room shows France's flag missions, never
  Germany's, and never a bomb. Each room's announcements begin where a full window of its own messages does, as the
  world's do. Until 2026-11-05 a flag's room had no announcements.
- **A stream is one room's.** `ListenForEvents` reads the identity token (`Identified`), as `GetHistory` does.
  `feed.Update.For(room)` says what goes to it: a message, or the tally of a message, of that room; an announcement to
  the world and to the room it was told in; a removal to every room, since it carries an id alone.
- **A pledge of another flag ends the stream of the room left.** `chat-rooms` hears `planet.v1.AllegiancePledged`, and
  `leave_room_usecase` publishes `feed.RoomLeft` with the room of the flag left. That account's stream of that room ends
  with `ErrNotInRoom`. A first flag leaves nothing. The client must not open it again: see its
  [Live chat](../../frontend/docs/chat.md).
- **Each room has its own seen mark**: `chat.seen` is keyed by `(account_id, room)`; `MarkSeen` and `GetHistory` take
  the room.
- **Mutes, the throttle, removals, reports and the retention are the same in every room.** `GetMessage` and
  `ListMessages` say a message's room (`flag_room`, empty for the world); `ListMessages` pages through every room in one
  list.
- **A room is not secret.** A new guest in a private tab can pick a flag and read its room.
- `room_test.go` pins the rule, the history query's suite the windows, the announcements and the marks of each room,
  and `listen_for_events_usecase`'s tests what each stream carries and when it ends.

**Chat has its own stream**, `ChatService.ListenForEvents` — see [The live streams](live-streams.md). It replaced a `/ws/chat` websocket that had to be kept apart from the tile one because frames carried a bare protobuf message with no type tag: a second payload on either socket would have been indistinguishable from the first. The `oneof` envelope is exactly what removes that constraint.

**Sending is an RPC, not a read on the socket**: both publishers lean on `CloseRead` for instant disconnect detection, and the RPC path already has the middleware stack and the interceptors.

`GetHistory` is marked `NO_SIDE_EFFECTS`, so Connect sends it as a GET — but it answers `Cache-Control: no-store`, the opposite of `GetMap`. A client fetches it once on join to seed what the stream then keeps up to date, so a cached answer would show a joiner a chat missing the last few minutes.
