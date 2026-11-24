# Support (`internal/support/`)

**What a player sends the operator, the operator's answers by email, and the player's replies.** A feature suggestion, a bug report, or a report of another player. Always on. It publishes one event, `support.v1.PlayerReported`, through its outbox, and handles one command, `support.v1.DecideReport`, both of a player report (below). It asks `auth` for the click token's key, and `player` how to name the accounts of a report when the operator reads it, and where to answer a note.

```
internal/support/internal/
  notes/                            Note (NewSuggestion, NewBugReport, Attaching, Anonymized, TakeSnapshot), Snapshot (Restore), Sent,
                                    ReplyAddress and the AddressBook port, Diagnostics,
                                    Attachment, AttachmentID (AttachmentIDOf), MediaType (MediaTypes), AttachmentRules, NoteID, Kind,
                                    AccountID (AccountIDOf), TextOf; the Storage, IDProvider and AttachmentIDProvider ports, the
                                    StorageContractSuite and SequentialIDs
    postgres_note_store/            Storage, over support.notes and support.attachments
    inmemory_note_storage/          Storage in a slice of snapshots, with Seed and Kept for a test — behind the testing tag, tests only
    s3_attachment_bucket/           the files notes carry, in an S3 bucket through cps3: upload links, view links, and what a file uploaded
                                    is. It signs its own links
    inmemory_attachment_bucket/     the same with no bucket, and Put for a file a test uploads — behind the testing tag, tests only
    uuid_id_provider/               IDProvider: a UUIDv7 per note; AttachmentIDProvider: a random UUID per file
    rpc_addresses/                  AddressBook: the account's address, from player.v1.InternalService/GetEmail
    mail_post/                      the operator's answer as a mail: from support.email, Reply-To the note's own address, threaded
    usecases/create_upload_usecase/     draws a file's id, checks it against the rules, keeps its room on the shelf,
                                        answers the link to put it at — AttachmentIDProvider, Shelf, Uploads
    usecases/suggest_feature_usecase/   draws an id, reads where to answer, reads the files named from the bucket, builds a
                                        suggestion with them and keeps it — IDProvider, Saver, Files, AddressBook
    usecases/report_bug_usecase/        the same, for a bug report — IDProvider, Saver, Files, AddressBook
    usecases/anonymize_notes_usecase/   reads the notes an account sent, anonymizes them, saves — Notes
    usecases/prune_usecase/             reads the notes past retention that still have a sender, anonymizes them, saves;
                                        Runner, which runs each prune — Notes
      log_prune/                        logs how many a prune went through
  conversations/                    Conversation (Answered, Replied, Closed, Anonymized, Thread, TakeSnapshot), Snapshot
                                    (Restore), Answer, Reply, ReplyTextOf; the Storage port and the StorageContractSuite
    postgres_conversation_store/    Storage, over support.note_answers, note_replies and note_closings
    inmemory_conversation_storage/  Storage in a slice of snapshots, with KeepNote, Seed and Kept — behind the testing tag
    usecases/answer_note_usecase/   reads the note and its conversation, has the note write the letter, mails it, keeps
                                    the answer — Notes, Conversations, Post
      audit_answer_note/            logs every answer at Warn
    usecases/close_note_usecase/    reads the conversation, closes it, saves — Conversations
    usecases/receive_reply_usecase/ the note the recipient names, the mail read, the reply kept without what it quotes, or
                                    stray — Inbox, Reader, Notes, Conversations
      log_receive_reply/            logs each mail: kept, kept already, stray (and why) or not kept
    mime_reader/                    Reader: a raw mail's Message-ID, From and words (the text part, or the HTML stripped)
  uploads/                          Upload (NewUpload, Discarded, TakeSnapshot), Snapshot (Restore), Shelf (Reserve, Room),
                                    FilesOf; the Storage port and the StorageContractSuite
    postgres_upload_store/          Storage, over support.uploads, and the attachments that name them
    inmemory_upload_storage/        Storage in a slice of snapshots, with Send and Kept for a test — behind the testing tag
    usecases/discard_unsent_usecase/    reads the files no note named in time, deletes them from the bucket, frees their
                                        room — Uploads, Bucket
  letters/                          Letter (Mail, Sent, Replied, Closed, Anonymized, Thread, TakeSnapshot), Draft (NewDraft,
                                    Mail, Sent), Snapshot (Restore), Mail, Sent, LetterID, SubmissionID, AddressOf and
                                    LetterOf (a letter's address under the inbox); the Storage and IDProvider ports, the
                                    StorageContractSuite
    postgres_letter_store/          Storage, over support.letters, letter_mails, letter_replies and letter_closings
    letter_post/                    a letter's mail: from support.email, Reply-To the letter's own address, threaded
    uuid_letter_ids/                IDProvider: a random UUID per letter
    usecases/write_letter_usecase/  the letter about a submission, to the account's address (AddressBook), or one
                                    more mail in a letter: mailed, then kept — IDProvider, AddressBook, Storage, Post,
                                    Clock
      audit_write_letter/           logs every letter at Warn
    usecases/close_letter_usecase/  reads the letter, closes it, saves — Storage
    usecases/receive_letter_reply_usecase/  the letter the recipient names, the mail read, the reply kept without
                                    what it quotes, or stray; ErrNotALetter before it reads a note's mail
      log_receive_letter_reply/     logs each mail
    usecases/anonymize_letters_usecase/ the letters to an account, anonymized — Storage
    usecases/prune_letters_usecase/ the letters written past retention that still name their player, anonymized
  reports/                          Report (NewReport, Forgotten, TakeSnapshot), Snapshot (Restore), Complaint, ReportID, Reason,
                                    MessageID; the Storage and IDProvider ports, the StorageContractSuite
                                    and SequentialIDs
    postgres_report_store/          Storage, over support.reports
    inmemory_report_storage/        Storage in a slice of snapshots, with Seed and Kept for a test — behind the testing tag, tests only
    uuid_id_provider/               IDProvider: a UUIDv7 per report
    usecases/report_player_usecase/     reads what was filed, draws an id, builds the report, keeps it and records
                                        support.v1.PlayerReported in one transaction — IDProvider, Transactioner,
                                        Reports, Outbox
    usecases/forget_reports_usecase/    reads the reports that name an account, forgets them, saves — Reports
    usecases/prune_reports_usecase/     reads the reports past retention, forgets them, saves — Reports
  decisions/                        Decision (NewDecision, TakeSnapshot), Snapshot (Restore), Outcome; the Storage port
    postgres_decision_store/        Storage, over support.report_decisions
    usecases/decide_report_usecase/ builds the decision and keeps it, or keeps it once (Once: a command's) — Saver
      audit_decide_report/          logs every decision at Warn
  supportv1controller/              SupportService and AdminService (bags), the interceptors
    suggest_feature_handler/  report_bug_handler/  report_player_handler/  get_attachment_rules_handler/
    create_upload_handler/  decide_report_handler/  answer_note_handler/  close_note_handler/
    list_notes_handler/notes_query/ PostgresQuery: ListNotesResponse straight from SQL, with a link to each file and the
                                    conversation, the open ones alone if asked — Links
    list_reports_handler/reports_query/   PostgresQuery: ListReportsResponse from SQL, decided or not, named — Players
  mailroomcontroller/receive_reply_handler/  POST /mailroom/replies on the admin listener: the mailroom Worker's mails
      rpc_players/                  Players, from player.v1.InternalService/GetPlayers, as player.v1.Player
    caller/                         the account on the context, or Unauthenticated
    pagecontext/                    Diagnostics: a request's PageContext and its User-Agent, for both handlers
    replyaddress/                   the address a note names, or the refusal with its NoteRefusal detail
    attachment/                     IDs and Uploaded: the files a note names, and the upload a request is answered
  commands/
    decide_report_handler/          support.v1.DecideReport → decide_report_usecase, once per report
  subscribers/                      Timeout
    player_forgotten_subscriber/    player.v1.PlayerForgotten → anonymize_notes_usecase, and again → forget_reports_usecase
    log_subscriber/                 logs an event a subscriber refused
  migrations/                       the support schema
```

- **A note is a text an account sent, and its kind**: `suggestion` or `bug`. Each use case draws the id, builds the note with `notes.NewSuggestion` or `notes.NewBugReport`, and saves it. The id is injected (`notes.IDProvider`), like the clock; tests use `notes.SequentialIDs`.
- **A note has no getter: it has a snapshot.** Nothing asks a note for a field. `notes.Snapshot` is a plain struct that mirrors its private fields and holds no rule. `Note.TakeSnapshot()` hands the state to a store. **`Snapshot.Restore()` is the one place a note's rules are checked**: an id, a known kind, a time, an address mail can go to or none (`ErrInvalidNote`), and a text that passes `TextOf` (`ErrInvalidText`). `NewSuggestion` and `NewBugReport` are a snapshot restored, so a note built and a note read back pass the same check. **A note sent has a sender**: `NewSuggestion` and `NewBugReport` refuse no account. A note read back may have none: it was anonymized. The stores, the test fake and the tests work on snapshots: a test compares `TakeSnapshot()` with a snapshot literal, and no constructor sits behind the `testing` tag. The operator's read does not go through `Note` at all (`notes_query`).
- **Every note carries its diagnostics**, `notes.Diagnostics`, a suggestion as a bug report: **a note has one shape, whatever its kind**. `kind` is a label and nothing depends on it: no field that only one kind has, and no rule in `Restore` that reads the kind. A thing that needed a field of its own would be a concept of its own, with its own table, as a player report is.
  - **What the page says of itself** is `PageContext`, in `SuggestFeatureRequest.context` and `ReportBugRequest.context`: the bundle it runs, the size of its viewport and its pixel ratio, whether it had lost the server, the graphics card as WebGL names it (`gpu_vendor`, `gpu_renderer`, `webgl_version`, `max_texture_size`), whether the globe had lost its WebGL context, and the cores and the memory the browser reports. **A page cannot read `chrome://gpu`**: the WebGL names are what it can read of it.
  - **The server reads the `User-Agent` itself.** `pagecontext.Diagnostics` is the one mapping, for both handlers.
  - It is plain data, a part of the snapshot, and `Restore` keeps it cleaned: no control character, 64 characters of build, 256 of user agent and of each graphics name, 128 of WebGL version, a viewport side of 100,000 at most, two decimals on a ratio or a memory, and 0 for what is not a number.
  - **Nothing in it is trusted**: the page writes it, so it helps read a note and decides nothing. A note whose page said nothing (an older client) has empty diagnostics, not none.
- **A note is anonymized, never deleted.** The text is worth keeping; who sent it is the personal part. `Note.Anonymized()` is the one change a note has: a copy with no account, and the same id, kind, text, diagnostics, files and time. **It keeps its diagnostics**: a user agent, a graphics card and a viewport with no account and no address describe a device, not a person. An anonymized note's account is zero in its snapshot, and `NULL` in `support.notes.account_id` (migration `20261012090000_note_sender_optional`).
- **A storage finds and saves, and holds no rule.** `notes.Storage` is two finders (`SentBy` an account, `WithSenderBefore` a time) and one `Save`, which upserts whatever notes it is handed. **It deletes nothing and reads no flag**: a change to a note is a method of `Note`, the use case reads, tells the note, and saves, and the adapter writes the state it is given without knowing why it changed. A finder restores each row it reads (`Snapshot.Restore()`), so a row that breaks a rule is an error, not a note.
- **A note keeps the address it is answered at** (`notes.Sent.ReplyAddress`, a part of the snapshot). See
  [The address a note is answered at](#the-address-a-note-is-answered-at).
- **`notes.TextOf` is the rule on the text**: valid UTF-8, and 1 to 1,000 runes (`MaxTextLength`) once trimmed. **Lines are kept**, which a chat message's are not: `\r\n` and `\r` become `\n`, a tab a space, and every other control character is dropped. The text goes to postgres and to the operator's JSON, so a newline forges nothing. A text that breaks the rule is `InvalidArgument` (`ErrInvalidText`).
- **The caller is the account in the click token**, a guest's included: `SuggestFeature`, `ReportBug` and `ReportPlayer` are `Attested` behind `supportv1controller.NewSessionInterceptor`, always enforcing (`support_session_checks{verdict}`). A token with no account is `Unauthenticated`.
- **One limiter for what is kept**, `support.rateLimiter`: three at once, then one a minute, per address (`cpipscope`), suggestions, bug reports and player reports together. `CreateUpload` has its own (see the files below).
- **Ten suggestions and bug reports a day, per account and per address** (`support.noteLimiter`): a call spends from the account's allowance and the address's together, or from neither (`supportv1controller.NewNoteLimitInterceptor`, `cpratelimit.Limiter.TakeAll`), so neither a new guest on one network nor one account on many networks sends more. It is a bucket of 10 refilled over a day, so a day is a rolling one. A player report is not a note: it has its own rule, once a day per player. **It is in memory**, like every limiter here: a restart forgets it. **The session check is outside the limiter**: the client answers a refused token by minting and sending again, and that first try must not cost one of the three. The order is the module's, so the smoke test `TestAGuestReportsABugWithAScreenshotAndTheOperatorOpensIt` pins it.
- **Kept in `support.notes`** (`id`, `kind`, `account_id`, `text`, `diagnostics`, `submitted_at`), one insert per note (5s timeout), in the module's own block, `support.database`. **`diagnostics` is `jsonb NOT NULL`** (migration `20261012100000_note_diagnostics`, which gives `{}` to the notes kept before it): every row has one, so no column is there for one kind alone. The store writes every key, and the keys are the JSON names of `PageContext` plus `userAgent`: `notes_query` decodes the object as a `PageContext` and skips a key it does not know, so a new diagnostic is a proto field and a key, and no migration. `migrations_test.go` starts from a note kept before the column. **`kind` is a postgres enum type, `note_kind`**: the database holds the list of kinds, so a row of another kind cannot exist, typed by hand or written by a bug. A new kind is a value a migration adds to the type, as `bug` is. **No IP address is kept**: the account is enough to find a player again, and the limiter holds the IP in memory only.
- **The account does not outlive its reason; a note's text does.** A note goes one way: read, `Anonymized()`, save. The prune reads the notes sent before `support.retention` (90 days) ago that still have a sender, 500 at a time until none is left, then forgets the reports filed before it (`prune_reports_usecase`), at boot and every `support.pruneInterval`; an anonymized note is not read again, which is what ends the loop. `player.v1.PlayerForgotten`, which player tells for every account auth deletes, reads the notes the account sent and anonymizes them at once (`support-notes-forgotten`), and forgets every report it filed or is named in (`support-reports-forgotten`): each concept has its own subscription. Until 2026-11-04 both heard `auth.v1.AccountDeleted` (`support-notes-accounts`, `support-reports-accounts`); why the new groups miss no deletion is in [Player](player.md#the-deploy-that-moved-the-others-to-players-events). **What stays of a note is free text and its files**: a player who wrote a name or an address in it, or showed one, left it there. The privacy page says both are kept and asks for no personal details in them.
- **The operator reads them with `support.v1.AdminService/ListNotes(since, kind, limit, open_only)`**, on the admin listener (see [Operator tools](admin.md)): newest first, 100 unless asked, 500 at most, every kind unless one is asked. Every note comes with its `context` and `user_agent`. It is a query (`notes_query`): a stored kind the wire cannot carry refuses the read (`ErrUnknownKind`: a kind a newer migration added to the type, read by an older build), and a kind asked for that the proto does not name is `InvalidArgument` (`ErrInvalidKind`). Each note carries its `account_id`, which is what `antibot.v1`'s `InspectPlayer` and `BanPlayer` and chat's `Mute` take; it is empty once the note is anonymized.
- **Nothing of a note is pushed.** No event tells a note, and nothing of one leaves the server but the files it
  carries, which go to the bucket: the operator asks.
- **A note carries up to 10 photos or videos**, a suggestion as a bug report: every note has the list, most of them empty. A `notes.Attachment` is an id, a media type and a size, plain data in the snapshot. The id (`notes.AttachmentID`) is the file's name in the bucket.
  - **The files are in an S3 bucket, never on this server.** The droplet has 4 GB of memory (1 GB until just before the season 0 finale) and 25 GB of disk, Cloudflare cuts a proxied body at 100 MB, Caddy passes Connect routes alone, and the operator reads on a loopback listener, so a file kept here could not be opened in a browser. Any S3-compatible bucket does (`support.attachments.bucket`); production's is Cloudflare R2.
  - **The files first, the note after**, so a note never reaches the operator without the files that explain it. `CreateUpload(content_type, size_bytes)` answers an id and a link: the page sends the file there with one `PUT`, straight to the bucket, and no byte of it passes through this server. Then `SuggestFeatureRequest.attachment_ids` and `ReportBugRequest.attachment_ids` name the files, in order.
  - **The server reads each file named before it keeps the note** (`Files.Attachments`, a presigned `HEAD` per file, 8 at once): its type and its size are what the bucket holds, never what the page says. A file that is not there is `notes.ErrNotUploaded`, answered `FailedPrecondition`, and nothing is kept. A type the list does not hold, or a size the rules do not take, is `InvalidArgument` as before: the rules are checked again on what was uploaded.
  - **The id is a random UUID** (`uuid_id_provider.NewAttachmentID`), not a v7: it is the only thing that lets a note name a file, so only the page that uploaded it knows it. A note naming the same id twice is refused.
  - **`CreateUpload` checks the rules first** (`Attaching` on one file): a type or a size they do not take gets no link. It is `Attested`, like a note, and has a daily limiter of its own, `support.uploadLimiter`: 100 files a day, per account and per address together, as the notes'.
  - **The bucket never holds more than `support.attachments.maxStoredBytes`** (9 GiB, under R2's free 10 GB): the bill is capped, not only slowed. **Every link signed takes its room first**, on the `uploads.Shelf`, whether its file comes or not, since a link cannot be taken back. `uploads.Upload` is that room: the id, the size the link signs, and when. `Shelf.Reserve` keeps it in `support.uploads` under one advisory lock for the whole table, so two links at once never both take the last of the room; a file that would not fit is `uploads.ErrShelfFull`, answered `ResourceExhausted`, and gets no link. A link can be used again until it expires, but on the same key and at the same size, so it never takes more room.
  - **The page is told the room left**: `GetAttachmentRules` answers `AttachmentRules.Within(room)`, no file bigger than what is left, and no file at all once the bucket is full, so the picker goes away rather than refuse.
  - **A file no note named in a day is deleted** (`support.attachments.keepUnsent`): the page was closed before Send, or the note was refused after its files went up. `discard_unsent_usecase` runs with the prune: it reads the uploads from before the cutoff that no row of `support.attachments` names (`Storage.Unsent`), deletes their files from the bucket (a signed `DELETE`, 8 at once; a file not there is deleted already), then marks them `Discarded`, and the store removes them, which frees their room. A file the bucket would not delete keeps its room, and the next prune tries again. **`keepUnsent` must outlast `uploadTTL`**, or the boot is refused: a file deleted while its link still works could come back with no room taken.
  - **`s3_attachment_bucket` signs each link itself**, through the shared `cps3` (AWS Signature Version 4, in the query): no SDK, and nothing is asked of the bucket to make one. An upload link signs the content type and the content length, so the bucket refuses another type or one byte more or less; it is good for `uploadTTL` (10m): the page asks for it right before it puts the file there. `cps3`'s `signature_test.go` signs the S3 reference's own example. The files are at the root of the bucket; the clips module's videos share it under `clips/videos/` (see [Clip videos](clip-videos.md)).
  - **The key is the attachment id**, and only the adapter knows it: the upload link, the read and the view link go through it, and `TestAFileGoesUpThroughItsLinkAndComesBackThroughItsViewLink` pins the first two against each other.
  - **`notes.AttachmentRules` is what a note sent may carry**: `Note.Attaching(files, rules)` refuses more files than `maxPerNote` (10) and a file over `maxBytes` (50 MiB), `ErrInvalidAttachment`, answered `InvalidArgument`. `Restore` only checks the media type and a size above 0, so a new limit never makes a kept note unreadable.
  - **`GetAttachmentRules` tells the page the rules**: the most files, the most bytes, and every content type. No token, no throttle, `no-store`. A server that takes none answers 0, and the page then shows no picker; a new limit needs no frontend release.
  - **The media types are a closed list** (`notes.MediaTypes`: JPEG, PNG, WebP, GIF, HEIC, HEIF, MP4, QuickTime, WebM), and the type `media_type` in postgres lists them again. A new one is a constant and a migration; the contract suite keeps a file of each, so a constant with no migration fails the tests. SVG is left out: it can run scripts.
  - **Kept in `support.attachments`** (`note_id`, `position`, `id`, `media_type`, `size_bytes`; migration `20261013100000_attachments`), written by the note store in the note's own transaction. It writes the list the snapshot holds and drops the rows past it, without knowing why the list changed.
  - **A note keeps its files for good, as it keeps its text.** A player sends them for the operator's use, so an anonymized note keeps them: only the account goes. Nothing deletes a file a note names. **Known risk**: a picture or a video can show who sent it (a username on screen, a voice), and nothing blanks that. The privacy page says the files are kept and asks for no personal details in them. Decided on 2026-10-14.
  - **Known risk: the bucket's request fees are not capped.** R2 charges each write past a million a month ($4.50 a million), and a link can be sent again and again until it expires. A short `uploadTTL` and the daily limits bound it; nothing in the app can stop it.
  - **Known risk: a full bucket stays full.** The files notes carry are kept for good, so once they reach `maxStoredBytes` no player can attach a file until the operator deletes some or raises the cap.
  - **The operator opens a file from `ListNotes`**: each note lists its attachments with `content_type`, `size_bytes` and `url`, a view link good for `linkTTL` (1h). `notes_query` reads the list in its one statement and asks its `Links` port, which the bucket fills, for each link.
  - **Off unless switched on** (`support.attachments.enabled`). Off, the rules take no file and the bucket block is not read: every link is `ErrNoBucket`. **Switch it off only once no note carries a file**: `ListNotes` fails on one that does.
  - **`s3_attachment_bucket`'s tests run on a real S3 server**, the Versity gateway in a container (MinIO's images are gone from the public registries): a file goes up and comes back, its type and size read back as uploaded, a file never uploaded is `ErrNotUploaded`, a file deleted is gone and the others stay, a link takes no other size or type, and a link past its time and a link signed with another secret are refused. `e2e/` uses a fake bucket that checks no signature.
- `notes.StorageContractSuite` runs on `inmemory_note_storage` and on postgres. It reads what a store kept, as snapshots, through a `Kept` hook each adapter's test fills, since the write side reads nothing back. Each snapshot read back must restore a note.

## The address a note is answered at

**The operator answers a note by email, so a note keeps the address it was sent with.**

- **The page sends one, or the account's is taken**: `SuggestFeatureRequest.reply_address` and
  `ReportBugRequest.reply_address` are `optional`. Given, it is read with `cpmail.AddressOf` (the rule auth reads an
  emailed code's address with, and no disposable-domain check), and one mail cannot go to is `InvalidArgument` with a
  `NoteRefusal` detail (`INVALID_REPLY_ADDRESS`, `supportv1controller/replyaddress`). Not given, support asks player
  for the account's (`notes/rpc_addresses`, `player.v1.InternalService/GetEmail`, which passes auth's on): **support
  never asks auth**. `notes.ReplyAddress` is the rule: the one given, or else the account's.
- **The page knows whether to ask**: auth's `GetMeResponse.has_email` says the account holds a verified address. A
  guest holds none, and is asked for one.
- **Every new note has one**: no address given and none on the account is `notes.ErrNoReplyAddress`, which
  `NewSuggestion` and `NewBugReport` refuse too, answered `FailedPrecondition` with a `NoteRefusal` detail
  (`NO_REPLY_ADDRESS`): the page shows its email field. The frontend that asks for one shipped first (see
  [A server restart reloads the page](../../../docs/reload-on-restart.md)).
- **Kept in `support.notes.reply_address`** (`text NOT NULL`, empty for none; migration
  `20261123115501_note_reply_address`): every note has the field. `Restore` checks a stored one with
  `cpmail.AddressOf`. The notes sent before it took their account's verified address once, on 2026-11-24, by an RPC
  removed after (#959): 45 filled, 3 left empty because a guest sent them. A note sent with no account has none.
- **`Anonymized()` blanks it with the account**: the prune at 90 days and `PlayerForgotten` take both. The privacy page
  says so.

## Answering a note

**The operator answers a note by email, and the player's replies come back to it.** What was said after a note is
sent is its conversation, a concept of its own (`conversations/`, which imports `notes`, never the other way): the
operator's answers, the player's replies, and when the operator last closed it with no answer. Player reports are not
answered.

- **A conversation has no getter: it has a snapshot**, as a note does. `Answered(text, mail, at)` adds an answer,
  `Replied(reply)` a reply (one whose Message-ID it holds already is `ErrReceivedAlready`), `Closed(at)` closes it,
  `Anonymized()` blanks each reply's sender, and `Thread()` is every mail's id, oldest first. `Restore` checks each
  mail's id (`cpmail.MessageIDOf`) and time, an answer's text (`notes.AnswerTextOf`: the rule of a note's text, up to
  5,000 characters) and cleans a reply's (`ReplyTextOf`: control characters dropped, lines kept, cut past 10,000
  characters, never refused).
- **Kept in three tables, one shape each** (migration `20261123124402_note_conversations`): `note_answers` and
  `note_replies`, each keyed by the note and the mail's Message-ID (a reply also keeps its `sender`, empty once
  anonymized), and `note_closings`, one row per note closed. An answer and a reply are two tables, not one table with a
  kind, since only a reply has a sender. Each goes with its note (`ON DELETE CASCADE`), though a note is never deleted.
- **Its storage finds and saves.** `Conversations(notes...)` answers one per note asked, in that order, empty for a
  note nobody wrote about; `Save` upserts every answer, reply and closing a snapshot holds, by key, and deletes
  nothing. A row about a note not kept is `notes.ErrUnknownNote` (the foreign key's refusal). An empty conversation
  writes nothing. `conversations.StorageContractSuite` runs on both stores.
- **`AdminService/AnswerNote(note_id, text)`** (`answer_note_usecase`): reads the note (`notes.Storage.Note`, or
  `NotFound`) and its conversation, has the note write the letter (`Note.Answer`: no address is `FailedPrecondition`, a
  text the rule refuses `InvalidArgument`), mails it (`mail_post`), then keeps the answer with the Message-ID
  Cloudflare answered, and answers when it was sent. **A mail that did not go keeps nothing** (`notes.ErrNotSent`,
  `Unavailable`): the page says so, and the operator sends again. A mail that went and could not be kept is the error
  net's `internal`; sent again, the player gets it twice. `audit_answer_note` logs each one at Warn, with the app that
  called (Claude's, through the connector). Its proto comment is its description as Claude's tool.
- **The letter** (`notes.Letter`, `Note.Answer`): to the note's address, subject by kind ("Re: your idea", "Re: your bug
  report"), plain text: the answer, then "On <the day>, you wrote:" and the note quoted line by line with `> `.
- **The mail** (`mail_post`): from `support.email` (`ClickPlanet <support@clickplanet.lol>`), **Reply-To the note's own
  address**, `support+<note id>@clickplanet.lol` (`notes.Inbox.Of`, which the reply's way in reads back with
  `NoteOf`), and, past the first, `In-Reply-To` the last mail of the thread and `References` every one, so the player's
  client keeps one thread. Through `cpmail`, with the token auth's codes use.
- **`AdminService/CloseNote(note_id)`** (`close_note_usecase`): the note leaves the open ones with no answer, until the
  player writes again. Closed again, the last time stays. `NotFound` for a note not kept.
- **`ListNotes` carries each note's `reply_address`, its `answers`, its `replies` (each with its sender) and
  `closed_at`**, read in its one statement. **`open_only` lists the notes waiting for an answer**: an address, and the
  player wrote last, the note or a reply after the last answer and the last close. The rule is the query's
  (`notes_query`): no command needs it.
- **Anonymized with its note**: `anonymize_notes_usecase` (`PlayerForgotten`) and the prune anonymize the conversations
  of the notes they read, then the notes, so a save that fails is read again: once a note has no sender, nothing finds
  it.
- **`support.email`** is a `cpmail.Config`: `delivery` (`log` unless set), `from` (`support@clickplanet.lol`),
  `fromName` (`ClickPlanet`), `cloudflare.accountId`, `cloudflare.apiToken`. A `from` that is not an address refuses
  the boot. Production sends through Cloudflare with auth's token (`env://CLOUDFLARE_EMAIL_TOKEN`); locally the answer
  is logged at Warn.

## Replies coming in

**A player's reply to an answer comes back to its note.** The answer's Reply-To is the note's own address,
`support+<note id>@clickplanet.lol`. Cloudflare Email Routing (Subaddressing on, one rule for `support@`) hands each mail
to it to the mailroom Worker (`apps/mailroom`, [The mailroom](../../mailroom/docs/mailroom.md)), a dumb pipe that posts
it raw. **All reading is here, in Go.**

- **The endpoint is `POST /mailroom/replies` on the admin listener** (`props.AdminHTTP`, as the connector's `/mcp`:
  a plain handler, outside the gate), which Caddy routes with no CORS (`@mailroom`). `receive_reply_handler`:
  - **`Authorization: Bearer <support.mailroom.secret>`**, compared in constant time (SHA-256 of each, then
    `subtle.ConstantTimeCompare`, so the length leaks nothing). None or another is a 401, and nothing is read.
  - **A size cap**, `support.mailroom.maxBytes` (25 MiB, Email Routing's own): a body past it is a 413, which the
    Worker turns into a rejection for good.
  - **The envelope recipient** is `X-Envelope-To`: the note's address, `+` part included.
  - **A 202 is kept** (`receive_reply_usecase.IsKept`: now, or already). **A 200 is a stray mail** (below), which the
    Worker forwards to the operator's mailbox. A mail not kept (the database) is a 500, and the Worker throws so the
    sending server tries again.
- **`mime_reader` reads the mail** (`net/mail`, `mime/multipart`): the Message-ID (`cpmail.MessageIDOf`; none, or not
  a mail at all, is `conversations.ErrUnreadable`), the From's bare address (empty if no mail goes back to it), and the
  words: the first `text/plain` part, else the first `text/html` with its tags, styles and quoted `<blockquote>` taken
  out. Parts marked as attachments are skipped (a reply's files are not kept), the transfer encodings
  (quoted-printable, base64) and the charsets (any WHATWG label, `golang.org/x/text`) decoded, 1 MiB read of a part,
  nested multiparts 5 deep at most.
- **`receive_reply_usecase`**: `notes.Inbox.NoteOf` reads the note id from the recipient, the note is read and must
  have an address (`Note.AddressError`), and the conversation keeps the reply (`Replied`) with **its quoted history
  cut** (`Inbox.Unquoted`: from the first line naming the inbox's domain, the quote's header as "On …, ClickPlanet
  <support+…@clickplanet.lol> wrote:", in any language, or Outlook's "From:", with the line before it when the header
  wraps; Outlook's separators; and the `>` lines at its end; lines a player answers between quotes stay), its From and
  the time it came.
- **Idempotent by its Message-ID**: a mail the conversation holds already is `ErrReceivedAlready`, done with. Two
  deliveries at once both write the same row (the key is the note and the Message-ID).
- **A stray mail goes to the operator's mailbox** (`receive_reply_usecase.Stray`, logged at Warn with why): a mail
  that cannot be read, a recipient that names no note (a player who wrote to `support@` itself), a note not kept, **a
  note with no address** (anonymized: its replies are not kept either). Nothing of it is kept here: the backend answers
  200, and the mailroom Worker forwards the mail as it came to its `OPERATOR_ADDRESS`. Why not keep it here for the
  backoffice is in [The mailroom](../../mailroom/docs/mailroom.md#why-a-forward-and-not-a-stray-mail-kept-in-the-backend).
  A kept one is logged at Info, one kept already at Info, one not kept at Error.
- **The page shows the From when it differs from the note's address**: `ListNotes` gives both.
- **No secret, no endpoint**: with `support.mailroom.secret` empty, nothing is mounted and no reply is taken. A secret
  of fewer than 32 characters refuses the boot.

## Writing to a player first

**The operator writes to a player first, by email, and the player's replies come back to it** (2026-11-24, for the
flags players submit: asking more, explaining a verdict). It is a **letter**, a concept of its own (`letters/`, which
imports `notes` and `conversations`, never the other way), not a kind of note: a note is what a player sent, and its
account is its sender, where a letter's is the player written to.

- **A letter is about one flag submission, and a submission has one letter at most** (2026-11-24): one conversation
  per submission, so a founder of Bavaria and Tyrol has two, and a reply about one is never read as the other's. The
  letter keeps the submission's id (`letters.SubmissionID`, `flags.v1.Submission.id`), its own type here: support
  imports nothing of the flags module, and does not check that the submission exists. **`NewDraft` decides the
  rule**: the use case hands it the letters kept about the submission (`Storage.About`), and one among them refuses
  the new one (`ErrWrittenAlready`, `AlreadyExists`). The unique index `letters_submission` holds it too: two first
  mails sent at the same moment both go, and the second is not kept (the store reads the index's refusal as
  `ErrWrittenAlready`). Accepted: one operator writes.
- **`AdminService/WriteLetter(letter_id, submission_id, account_id, subject, text)`** (`write_letter_usecase`): with
  no `letter_id`, the letter about the submission (none is `ErrInvalidSubmission`, `InvalidArgument`): the account's
  verified address (`notes.AddressBook`, player's `GetEmail`; none is `ErrNoReplyAddress`, `FailedPrecondition`), a
  subject of 1 to 200 characters (`ErrInvalidSubject`; the backoffice sets "Your flag <name>"), the text by the rule
  of an answer (`notes.AnswerTextOf`). With one, one more mail in that letter, under "Re: " its subject, in its
  thread. **Mailed, then kept**, as an answer: a mail that did not go keeps nothing (`Unavailable`), one that went and
  could not be kept would go twice if sent again. `audit_write_letter` logs each at Warn, with the app that called.
- **The mail** (`letter_post`): from `support.email`, **Reply-To the letter's own address**,
  `support+l-<letter id>@clickplanet.lol` (`letters.AddressOf`), and past the first, `In-Reply-To` and `References`
  (`mail_post.Threaded`, the notes'). The `l-` tag means a letter's address never reads as a note's.
- **A reply comes in as a note's does**: the mailroom posts it to `/mailroom/replies`, and the handler asks the letters
  first (`receive_letter_reply_usecase`: `letters.LetterOf` is `ErrNotALetter` before it reads anything for a note's
  address), then the notes. The reply is kept without what it quotes (`Inbox.Unquoted`), once per Message-ID. A letter
  not kept or anonymized is stray: the Worker forwards the mail to the operator.
- **`AdminService/ListLetters(account_id, submission_id, open_only, limit)`** (`letters_query`): the latest written or
  replied to first, 100 unless asked, 500 at most, with their mails and replies, each with its `submission_id`;
  `account_id` for one player's, `submission_id` for the one about a submission. **`open_only`**:
  a reply after the last mail, and no close after it. **`CloseLetter(letter_id)`**: the letter waits for nothing until
  the player writes again.
- **Kept in four tables, one shape each** (migration `20261124103252_letters`): `letters` (`submission_id`, unique;
  `account_id` null and `address` empty once anonymized, `subject`, `written_at`, its first mail's), `letter_mails`,
  `letter_replies` (with `sender`, empty once anonymized) and `letter_closings`, each keyed by the letter and the mail.
  Migration `20261124125235_letter_submissions` added `submission_id` and deleted the letters written before it, on
  their first day, which named no submission. A reply to one comes in stray, and the Worker forwards it.
- **Anonymized as notes are**: `player.v1.PlayerForgotten` (`support-letters-forgotten`) and the prune, past
  `support.retention` from the first mail (`prune_letters_usecase`, in the prune's runner), blank the account, the
  address and each reply's sender. What was written stays, and so does the submission it is about.

**A report is a concept of its own** (`reports/`), not a third kind of note: it names a second account, it has a reason, and it can be abused. `reports` imports `notes` (the account, the rule on a text), never the other way.

- **`ReportPlayer(account_id, reason, text, message_id)`**. **A report names the account, never a name.** A name is unique at one instant, but a rename frees it and another account can take it, so a report by name could land on the wrong player. The client holds the id: it is on each line a card opens from (`ChatMessage.author_account_id`, `RosterEntry.account_id`, `Standing.account_id`). The handler parses it (`notes.AccountIDOf`); what is not an account id is `InvalidArgument`. **Nothing checks that the account exists**: the module does not know which accounts do, and a report of nobody is a row the operator reads with no name. Player's `TestARenamedPlayerIsNamedByItsNewNameAndNotByWhoTookItsOldOne` pins the reason.
- **A report is kept as two accounts, not as a copy of a name**, as a chat message is. So a rename does not lose who was reported, and `ListReports` names both as the game shows them when it is read.
- **A report has no getter either: it has a snapshot**, as a note does. `reports.Snapshot` mirrors its fields; `Report.TakeSnapshot()` hands them to a store; **`Snapshot.Restore()` is the one place the rules of one report are checked**, and `NewReport` is a snapshot restored, then checked against the reports before it.
- **Its storage finds and saves**, as a note's does. `reports.Storage` is three finders (`FiledBy` a reporter about a player, `Naming` an account, `FiledBefore` a time) and one `Save` that upserts what it is handed. A finder restores each row it reads.
- **A report is forgotten, where a note is anonymized.** `Report.Forgotten()` marks it, and `Save` removes the row of a forgotten report: nothing of it is kept. A report names two accounts and its text is about the second one, so blanking an account would leave a text about a player. This is where the two stores differ: a note's store deletes nothing.
- **What the reporter says is `reports.Complaint`**: plain data, a part of the snapshot, which the handler fills from the request. `Restore` checks it: the reason is one of `cheating`, `chat_abuse`, `bad_name` and `other` (`ErrUnknownReason`; the proto's `ReportReason` is the list, and the handler maps it); the text is optional, and a text given follows `notes.TextOf`; the message id is the chat's, cut of control characters, and dropped past 64 characters. **The message id is not checked against the chat**: it is a pointer for the operator, who reads the message there.
- **The use case only orders four steps**: the read of what the reporter already filed about that account, the id, then `NewReport` and the save. A report's rules live in `Restore` alone, the complaint's included.
- **No report of oneself** (`ErrOwnAccount`, `InvalidArgument`): `Restore` refuses it, so no report of oneself exists to be kept.
- **One account reports one player once a UTC day** (`ErrReportedAlready`, `AlreadyExists`), **and `NewReport` is where that is decided**: the use case hands it what the reporter already filed about the player (`FiledBy`), and a report of the same day among them refuses the new one. **The database holds no part of the rule**: no day column, no unique constraint, and a storage keeps two reports of one day if it is handed them. So two reports sent at the very same moment can both be kept. That is accepted: a report does nothing by itself, and the service's limiter lets an address send three.
- **A report does nothing by itself, and nothing counts reports.** No mute, no ban, no score. A guest account is free to make and countries are rivals here, so a threshold would be a weapon. The operator reads a report, then uses the tools it already has; the chat's judge reads it too (below), and acts by what was said, never by how many reported it.
- **Every report kept is told**: `support.v1.PlayerReported` (`proto/support/v1/events.proto`: the report's id, both
  accounts, the reason as the wire names it, the text, the message id and the time). `report_player_usecase` records it
  in its outbox in the transaction that saves the report, so the chat is never told of a report not kept, nor misses
  one kept (migration `20261107130000_outbox`, `support-outbox`, the relay in the module's runners). **The chat hears
  it**: its judge reads a report of what was said, and closes it with a command (see
  [Chat moderation](../../../docs/moderation.md)). Support is upstream of the chat, and never names it.
- **A report is closed by a command too**: `support.v1.DecideReport{report_id, outcome, comment}`
  (`proto/support/v1/commands.proto`, on its own command stream, group `handler`), which `decide_report_handler`
  hands to `decide_report_usecase` with `Once`. **The report's id is the command's identity**: the store keeps it with
  `SaveOnce` (`ON CONFLICT DO NOTHING`), so a report decided already keeps its decision, the operator's or the same
  command's told again, and the command is done (`ErrDecidedAlready`). The operator's `DecideReport` call still
  replaces any decision. A report no longer kept is nothing to close, and the command is done too. A command with no
  report id or no outcome is refused and logged, then buried once the bus gives up.
- **Kept in `support.reports`** (`id`, `reporter_account_id`, `reported_account_id`, `reason`, `text`, `message_id`, `created_at`; migration `20261012110000_reports`), with no address. **`reason` is a postgres enum type, `report_reason`**, as a note's kind is: a new reason is a value a migration adds to the type.
- **`AdminService/ListReports(since, limit)`**, as `ListNotes`: newest first, 100 unless asked, 500 at most. `reports_query` reads the rows, then asks `GetPlayers` once for everyone they name, reporters and reported together. An account nobody can name any more has an empty name. **Reports nobody could be named in are refused**, as a chat history is. A stored reason the wire cannot carry refuses the read (`ErrUnknownReason`: a reason a newer migration added to the type, read by an older build). `reported_account_id` is what `antibot.v1`'s `InspectPlayer` and `BanPlayer` and chat's `Mute` take, and `message_id` what `GetMessage` and `RemoveMessage` take.
- **`AdminService/DecideReport(report_id, outcome, comment)` closes a report**: `acted` or `dismissed`, and an
  optional comment that follows `notes.TextOf`. **A decision is a concept of its own** (`decisions/`), in
  `support.report_decisions` (migration `20261018130000_report_decisions`, `outcome` the type `report_outcome`), one
  row per report: a second decision replaces the first, and `audit_decide_report` logs each one. **It goes with its
  report** (`ON DELETE CASCADE`), so a forgotten report takes its decision. A report no longer kept is `NotFound`
  (the store reads the foreign key's refusal as `decisions.ErrUnknownReport`).
- **`ListReports` carries each decision** (`Report.decision`, unset while open), and `open_only` leaves out the
  decided ones. A stored outcome the wire cannot carry refuses the read (`reports_query.ErrUnknownOutcome`).
- With no internal listener the boot is refused: the module dials it once, at build, for `player`, which names the accounts in `ListReports`.
- `reports.StorageContractSuite` runs on `inmemory_report_storage` and on postgres, through the same `Kept` hook of snapshots.
