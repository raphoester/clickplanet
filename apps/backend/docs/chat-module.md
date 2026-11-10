# Inside the chat module: the same shape

Chat follows the same rules as planet: no `domain`, no `adapters`, one directory per concept. It has ten:
`messages`, `reactions` (which imports `messages`), `announcements` (which imports `messages`, for the room it is told
in), `seen` (which imports `messages`), `mutes`
(which imports `messages`), `removals` (which imports `messages`), `moderation`, the history of what an operator
did to an account (which imports `messages` and `mutes`), `raids`, the raid mode that closes the chat to guests (which
imports `messages` and `moderation`), `judgements`, the judge of the reports support tells (which imports `messages`,
`moderation` and `mutes`), and `feed`, the live stream, which carries the
first three and the removals. `subscribers/` is its edge for events, as `chatv1controller/` is
its edge for the wire. A read is a query under its handler: see [Reads are queries](queries.md).

```
internal/chat/internal/
  messages/                             Message (Snapshot, Restore, TakeSnapshot, CheckSender, In, Answering), MessageID, Quote,
                                        Drafts, Thread, Record (RecordSnapshot), Limits (WithBanned), Banned (a
                                        cpphrases.List: NewBanned, DefaultBannedPhrases), ErrBanned, AccountID, Author,
                                        Room (World, FlagRoom, RoomOf), Rooms (EntryError), Showing,
                                        ErrNoAccount, ErrNotShown, ErrNotKept, ErrNotInRoom, Window, the Storage, Flags and
                                        IDProvider ports, the StorageContractSuite, SequentialIDs, Restored and
                                        FakeFlags
    postgres_message_store/             Storage, over chat.messages; KeptMessage, a message removed or not, for a removal
    inmemory_message_storage/           Storage in a slice — behind the testing tag, tests only
    rpc_player_authors/                 who posts, from player.v1.InternalService/IntroducePlayer, on each post: a
                                        guest or not, and the name a mute line keeps
    rpc_planet_flags/                   Flags: the flag an account plays for, from
                                        planet.v1.InternalService/GetAllegianceOf, on each call to a flag's room
    log_authors/                        logs a caller it could not introduce
    uuid_id_provider/                   IDProvider: a random UUID, as text
    usecases/send_message_usecase/      drafts, replies, names, appends, publishes — Appender, Publisher, Authors, Drafts,
                                        Thread
      publishing_send_message/          publishes chat.v1.MessageSent once a message is kept
      muting_send_message/              refuses a muted caller before anything else, and a GIF to one muted from
                                        GIFs — Mutes
      log_banned/                       logs a post refused for a banned phrase, with the phrase as listed
      raiding_send_message/             refuses a guest's post while raid mode runs, and asks who posts only then —
                                        RaidMode, Authors
    usecases/prune_usecase/             deletes past retention, from each table, each set of tables past its own
                                        (KeptFor: the chat's, the moderation history's); Runner — Pruner
      log_prune/                        logs what a prune deleted
  reactions/                            Reaction, Reactor, AccountOf, Reactions, Count, Tally, Change, the Storage port
                                        and its suite
    postgres_reaction_store/            Storage, over chat.reactions
    inmemory_reaction_storage/          Storage in a slice — behind the testing tag, tests only
    usecases/react_usecase/             puts a reaction on or off, publishes the tally — Messages, Board, Publisher
      muting_react/                     refuses a muted caller before anything else — Mutes
  announcements/                        Announcement (its Room, WithRoom, In), AnnouncementID, Kind (Kinds, Known:
                                        announce_usecase refuses any other), Bomb, Fortify, Muted, Round, Champion
                                        and FlagMission: each builds its Announcement, named after the fact it tells;
                                        RoundID and BookID, scoring's ids; FlagMissionID, missions'; the Storage port
                                        and the suite; FakeResults behind the testing tag
    postgres_announcement_store/        Storage, over chat.announcements
    inmemory_announcement_storage/      Storage in a slice — behind the testing tag, tests only
    usecases/announce_usecase/          keeps an announcement once, publishes it the first time — Appender, Publisher
    usecases/announce_mute_usecase/     names a muted account, announces it as a Muted — Authors, Announcer
    usecases/announce_round_usecase/    reads a round's results by id, announces its podium — Results, Announcer
    usecases/announce_champion_usecase/ reads a closed book's table by id, announces its champion — Results, Announcer
    rpc_scoring_results/                Results, from scoring.v1.InternalService/GetRound and GetTable
  seen/                                 Until (the rule on a mark's time), ErrNoTime, the Storage port: writes only
    postgres_seen_store/                Storage, over chat.seen; what it keeps is tested through history_query
    inmemory_seen_storage/              Storage in a map, and Kept for a test — behind the testing tag, tests only
    usecases/mark_seen_usecase/         keeps until when an account saw the chat — Saver
    usecases/forget_seen_usecase/       deletes an account's mark — Deleter
  mutes/                                Mute (NewMute, MuteOf, Ended, Lifted, ApplicableTo, Refusal), MuteID, Kind (Chat,
                                        Gifs), Caller, Scope (ScopeOf), DurationOf, Book (MuteError), the IDProvider
                                        and Storage ports, the suite
    postgres_mute_store/                Storage, over chat.mutes
    inmemory_mute_storage/              Storage in a slice — behind the testing tag, tests only
    uuid_id_provider/                   IDProvider: a UUIDv7 per mute
    proto_kind/                         Encode and Decode: a Kind as chat.v1.MuteKind, for the admin calls, the
                                        refusal and chat.v1.AccountMuted
    usecases/mute_usecase/              keeps a mute on the account and its network, and records it in the history, in
                                        one transaction — Saver, Recorder, Addresses, IDProvider
      publishing_mute/                  publishes chat.v1.AccountMuted, with its kind, once a mute is kept
      audit_mute/                       logs every mute at Warn
    usecases/unmute_usecase/            ends now every running mute of a kind given to the account, and records it —
                                        Transactioner, Store, Recorder
      audit_unmute/                     logs every unmute at Warn
  removals/                             Removal (NewRemoval, TakeSnapshot), Snapshot, ErrUnknownMessage, the Storage port
    postgres_removal_store/             Storage, over chat.removals
    usecases/remove_message_usecase/    takes a kept message out of what anybody is shown, records it with a copy of
                                        what it said, then tells the streams — Messages, Saver, Recorder, Publisher
      audit_remove/                     logs every removal at Warn
  moderation/                           Action (NewMute, NewUnmute, NewRemoval, TakeSnapshot), Snapshot, Kind (Muted,
                                        Unmuted, Removed), Operator, Reason (ReasonOf, ErrInvalidReason), the Storage port:
                                        writes only
    postgres_moderation_store/          Storage, over chat.moderation; ActionsOn, what the judge counts the steps from
    inmemory_moderation_storage/        Storage in a slice, and Kept for a test — behind the testing tag, tests only
    usecases/forget_moderation_usecase/ deletes the actions taken on an account, takes it off the ones it took — Forgetter
  raids/                                RaidMode (Start, Ended, RunsAt, Admits, TakeSnapshot), Snapshot (Restore),
                                        DurationOf, MaxDuration, Book (Running), ErrGuest, ErrNoRaidMode, the Storage
                                        port and its suite
    postgres_raid_store/                Storage, over chat.raid_mode: one row at most
    inmemory_raid_storage/              Storage in memory, and FailWith — behind the testing tag, tests only
    usecases/set_raid_mode_usecase/     starts raid mode from now, or ends the one that runs — Store
      audit_set_raid_mode/              logs every call at Warn
  judgements/                           Report (ReportID, Reason), Line, Conversation, Answer (Category, Spend), Slate
                                        (SlateOf, Next), Step, Decide, Decision (Decided), Judgement (Verdict, Followed,
                                        Comment), Snapshot (Restore), Client and By ("Claude judge"), ErrNoJudge,
                                        ErrUnjudgeable; the Storage port and its suite
    postgres_judgement_store/           Storage, over chat.judgements: one per report, the first that judged a message
    inmemory_judgement_storage/         Storage in a slice, and Kept and FailWith — behind the testing tag, tests only
    postgres_conversations/             the message reported, its room around it and the player's own latest lines
    claude_judge/                       Judge: asks Claude Haiku, with instructions.md and the policy, for an Answer held
                                        to a schema; Unkeyed with no key
    redis_outbox_sender/                what the chat's outbox relay sends: support.v1.DecideReport as a command,
                                        chat.v1.AccountMuted as an event
    usecases/judge_report_usecase/      reads, asks, then removes, mutes and decides in one transaction — Judgements,
                                        Conversations, Authors, History, Judge, Calls, Transactioner, Remover, Muter,
                                        Outbox
      log_judgement/                    logs every judgement and its cost, at Warn what the operator should see
  feed/                                 Update: a message sent, a message's new reactions, an announcement, a
                                        message removed, or an account that left a flag's room (MessageSent,
                                        ReactionsChanged, Announced, MessageRemoved, RoomLeft); For(room)
    inprocess_feed/                     the fanout to every open stream, in this process
    usecases/listen_for_events_usecase/ one client's feed of one room, heartbeat   — UpdatesSubscriber, Rooms
    usecases/leave_room_usecase/        ends an account's streams of the room it left — Publisher
  chatv1controller/                     ChatService and AdminService (bags), the interceptors
    send_message_handler/  get_history_handler/  listen_for_events_handler/  react_handler/  mark_seen_handler/
    mute_handler/  unmute_handler/  get_mute_handler/  remove_message_handler/  get_message_handler/
    list_messages_handler/  list_moderation_handler/  set_raid_mode_handler/  get_raid_mode_handler/
    get_mute_handler/mute_query/        PostgresQuery: the mute that holds an account now, by itself or its network
    list_moderation_handler/moderation_query/  PostgresQuery: the moderation history, newest first, one account's or
                                        everybody's, since a time
    get_message_handler/message_query/  PostgresQuery: one message as it was sent, removed or not
    get_raid_mode_handler/raid_mode_query/  PostgresQuery: the raid mode that runs now, if one does
    list_messages_handler/messages_query/  PostgresQuery: a page of messages as they were sent, before or after a
                                        message's id, named for the operator — Authors
      rpc_player_authors/               Authors, from player.v1.InternalService/GetPlayers, as player.v1.Player
    get_history_handler/history_query/  PostgresQuery: GetHistoryResponse straight from SQL, by account, and until
                                        when the viewer saw the chat
    chatmessage/                        Encode, EncodeCounts and Reaction (the wire's enum, checked), for the post,
                                        the reaction and the stream
    chatannouncement/                   Encode, for the stream
    chatmute/                           Refusal: a mute as PermissionDenied with a MuteRefusal detail
    chatoperator/                       Of: who calls an admin service, off the context the admin gate filled
  subscribers/                          Timeout
    bomb_landed_subscriber/             planet.v1.BombLanded → announce_usecase, as a Bomb
    fortified_subscriber/               planet.v1.Fortified → announce_usecase, as a Fortify of 50 tiles or more
    account_muted_subscriber/           chat.v1.AccountMuted → announce_mute_usecase
    day_ended_subscriber/               gamemaster.v1.DayEnded → announce_round_usecase, as a DayOf
    battle_ended_subscriber/            gamemaster.v1.BattleEnded → announce_round_usecase, as a BattleOf
    season_ended_subscriber/            gamemaster.v1.SeasonEnded → announce_champion_usecase
    flag_mission_done_subscriber/       missions.v1.FlagMissionDone → announce_usecase, as a FlagMission, in the
                                        flag's room and the world
    player_forgotten_subscriber/        player.v1.PlayerForgotten → forget_seen_usecase, and forget_moderation_usecase
    allegiance_pledged_subscriber/      planet.v1.AllegiancePledged → leave_room_usecase, for the flag left
    player_reported_subscriber/         support.v1.PlayerReported → judge_report_usecase
    log_subscriber/                     logs an event a subscriber refused (player's, copied)
  migrations/                           the chat schema
```

- **The rules that need no port are in the `messages` root**: `Limits` (runes, UTF-8, control characters, and the
  banned phrases, `Banned`), `AccountIDOf` (the context's account, or `cpsession.NoAccount`) and `NewRecord` (truncates the user agent). They are tested there, not through the use case.
- **Ids are injected** (`messages.IDProvider`, `mutes.IDProvider`), like the clock. Tests use each concept's `SequentialIDs` (behind the tag), so they assert exact ids. An announcement draws none: its id is the fact's (see [Announcements](chat.md#announcements)). `messages.Drafts` holds the ids, the clock, the `Limits` and the country list, and builds a new `Message` from what was sent; `messages.Thread` holds what a reply needs (see [Replies](chat.md#replies)). So `send_message_usecase` stays at five ports. A message id is still a UUID string: `chat.messages.id` is text, with no unique constraint.
- **`send_message_usecase.Config` stays under `chat.Config.Service`**, so the `chat.service.*` keys do not change.
- **`chatv1controller`'s root tests are about the chain** (error net, blocklist, throttle). Each handler package tests its own mapping.
