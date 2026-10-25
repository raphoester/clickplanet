# Inside the chat module: the same shape

Chat follows the same rules as planet: no `domain`, no `adapters`, one directory per concept. It has seven:
`messages`, `reactions` (which imports `messages`), `announcements`, `seen` (which imports `messages`), `mutes`
(which imports `messages`), `removals` (which imports `messages`), and `feed`, the live stream, which carries the
first three and the removals. `subscribers/` is its edge for events, as `chatv1controller/` is
its edge for the wire. A read is a query under its handler: see [Reads are queries](queries.md).

```
internal/chat/internal/
  messages/                             Message (Snapshot, Restore, TakeSnapshot, Named, Answering), MessageID, Quote,
                                        Drafts, Thread, Record (RecordSnapshot), Limits, AccountID, Author, Title, Rank,
                                        ErrNoAccount, ErrNotShown, Window, the Storage and IDProvider ports, the
                                        StorageContractSuite, SequentialIDs and Restored
    postgres_message_store/             Storage, over chat.messages
    inmemory_message_storage/           Storage in a slice — behind the testing tag, tests only
    rpc_player_authors/                 who an account is, from player.v1.InternalService/GetAuthor(s): one on
                                        each post, the people under a message on each reaction
    log_authors/                        logs a caller, or a page of them, it could not name
    uuid_id_provider/                   IDProvider: a random UUID, as text
    usecases/send_message_usecase/      drafts, replies, names, appends, publishes — Appender, Publisher, Authors, Drafts,
                                        Thread
      publishing_send_message/          publishes chat.v1.MessageSent once a message is kept
      muting_send_message/              refuses a muted caller before anything else — Mutes
    usecases/prune_usecase/             deletes past retention, from each table; Runner — Pruner
      log_prune/                        logs what a prune deleted
  reactions/                            Reaction, Reactor, AccountOf, Reactions, Count, Tally, Change, Named,
                                        AccountsOf, the Storage port and its suite
    postgres_reaction_store/            Storage, over chat.reactions
    inmemory_reaction_storage/          Storage in a slice — behind the testing tag, tests only
    usecases/react_usecase/             puts a reaction on or off, publishes the tally — Messages, Board, Publisher, Authors
      muting_react/                     refuses a muted caller before anything else — Mutes
  announcements/                        Announcement, AnnouncementID, Kind (Kinds, Known: announce_usecase refuses
                                        any other), Bomb, Muted and Round (payloads), Champion (its own Announcement,
                                        under its season's id), the Storage and IDProvider ports, the suite and
                                        SequentialIDs
    postgres_announcement_store/        Storage, over chat.announcements
    inmemory_announcement_storage/      Storage in a slice — behind the testing tag, tests only
    uuid_id_provider/                   IDProvider: a random UUID
    usecases/announce_usecase/          keeps an announcement, then publishes it — Appender, Publisher, IDProvider
    usecases/announce_mute_usecase/     names a muted account, announces it as a Muted payload — Authors, Announcer
    usecases/announce_champion_usecase/ keeps a season's champion once, publishes it the first time — Appender,
                                        Publisher
  seen/                                 Until (the rule on a mark's time), ErrNoTime, the Storage port: writes only
    postgres_seen_store/                Storage, over chat.seen; what it keeps is tested through history_query
    inmemory_seen_storage/              Storage in a map, and Kept for a test — behind the testing tag, tests only
    usecases/mark_seen_usecase/         keeps until when an account saw the chat — Saver
    usecases/forget_seen_usecase/       deletes an account's mark — Deleter
  mutes/                                Mute (NewMute, MuteOf, ApplicableTo, Refusal), MuteID, Caller, Scope (ScopeOf),
                                        DurationOf, Book (MuteError), the IDProvider and Storage ports, the suite
    postgres_mute_store/                Storage, over chat.mutes
    inmemory_mute_storage/              Storage in a slice — behind the testing tag, tests only
    uuid_id_provider/                   IDProvider: a UUIDv7 per mute
    usecases/mute_usecase/              keeps a mute on the account and its network — Saver, Addresses, IDProvider
      publishing_mute/                  publishes chat.v1.AccountMuted once a mute is kept
      audit_mute/                       logs every mute at Warn
  removals/                             Removal (NewRemoval, TakeSnapshot), Snapshot, ErrUnknownMessage, the Storage port
    postgres_removal_store/             Storage, over chat.removals
    usecases/remove_message_usecase/    takes a kept message out of what anybody is shown, then tells the streams —
                                        Messages, Saver, Publisher
      audit_remove/                     logs every removal at Warn
  feed/                                 Update: a message sent, a message's new reactions, an announcement, or a
                                        message removed (MessageSent, ReactionsChanged, Announced, MessageRemoved)
    inprocess_feed/                     the fanout to every open stream, in this process
    usecases/listen_for_events_usecase/ one client's feed, heartbeat   — UpdatesSubscriber
  chatv1controller/                     ChatService and AdminService (bags), the interceptors
    send_message_handler/  get_history_handler/  listen_for_events_handler/  react_handler/  mark_seen_handler/
    mute_handler/  remove_message_handler/  get_message_handler/  list_messages_handler/
    get_message_handler/message_query/  PostgresQuery: one message as it was sent, removed or not
    list_messages_handler/messages_query/  PostgresQuery: a page of messages as they were sent, before or after a
                                        message's id, named — Authors
      rpc_player_authors/               the same adapter as the history's, for this query
    get_history_handler/history_query/  PostgresQuery: GetHistoryResponse straight from SQL, named, and until when
                                        the viewer saw the chat — Authors
      rpc_player_authors/               Authors, from player.v1.InternalService/GetAuthors, as player.v1.Author
    chatmessage/                        Encode, EncodeCounts and Reaction (the wire's enum, checked), for the post,
                                        the reaction and the stream
    chatannouncement/                   Encode, for the stream
    chatmute/                           Refusal: a mute as PermissionDenied with a MuteRefusal detail
  subscribers/                          Timeout
    bomb_landed_subscriber/             planet.v1.BombLanded → announce_usecase, as a Bomb payload
    account_muted_subscriber/           chat.v1.AccountMuted → announce_mute_usecase
    round_closed_subscriber/            seasons.v1.RoundClosed → announce_usecase, as a Round payload
    season_closed_subscriber/           seasons.v1.SeasonClosed → announce_champion_usecase
    account_deleted_subscriber/         auth.v1.AccountDeleted → forget_seen_usecase
    log_subscriber/                     logs an event a subscriber refused (player's, copied)
  migrations/                           the chat schema
```

- **The rules that need no port are in the `messages` root**: `Limits` (runes, UTF-8, control characters), `AccountIDOf` (the context's account, or `cpsession.NoAccount`) and `NewRecord` (truncates the author id and user agent). They are tested there, not through the use case.
- **Ids are injected** (`messages.IDProvider`, `announcements.IDProvider`, `mutes.IDProvider`), like the clock. Tests use each concept's `SequentialIDs` (behind the tag), so they assert exact ids. `messages.Drafts` holds the ids, the clock, the `Limits` and the country list, and builds a new `Message` from what was sent; `messages.Thread` holds what a reply needs (see [Replies](chat.md#replies)). So `send_message_usecase` stays at five ports. A message id is still a UUID string: `chat.messages.id` is text, with no unique constraint.
- **`send_message_usecase.Config` stays under `chat.Config.Service`**, so the `chat.service.*` keys do not change.
- **`chatv1controller`'s root tests are about the chain** (error net, blocklist, throttle). Each handler package tests its own mapping.
