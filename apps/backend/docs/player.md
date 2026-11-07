# Player (`internal/player/`)

**Who an account is to the other players: the name it chose or its guest code, its color, whether it is an admin of the game, and who is playing now; an account asked deleted, hidden from the others until it is purged or comes back; and the names an operator gave it.** It makes no account and mints nothing. Always on: the chat asks it who posts. Its only upstream is `auth`, and it tells the other modules what they need of an account in its own words (below), so none of them needs to hear auth.

**What an account did and earned is [Progression](progression.md)**: the stats, the streaks, the titles and the title worn, the fronts, the badges and the season finales fought were player's until 2026-11-04. They moved out because they made player hear the whole game, while the whole game asks player who a player is: see [Why a module of its own](progression.md#why-a-module-of-its-own).

```
internal/player/internal/
  players/                          AccountID, Account (AccountOf), Name (NameOf), GuestCode (GuestCodes, DisplayNameOf), Player (PlayerOf),
                                    Tag (TagOf), Profile (ProfileSnapshot, Unnamed, Renamed, RenamedByOperator), NameChoice (the rename
                                    rule), Color; the Store port and its contract suite
    postgres_player_store/          the Store over player.profiles and player.guest_codes
    random_code_generator/          draws guest codes from crypto/rand
    random_name_generator/          draws a username for a signed-in account: an adjective, an animal, two digits
    inmemory_player_store/          the same port in maps, behind the testing tag
    rpc_account_reader/             whether an account is linked and when it was made, from auth.v1.InternalService/GetAccount,
                                    and the same of a page at once, from GetAccounts (players.Account)
    usecases/set_name_usecase/  set_color_usecase/  introduce_player_usecase/
      set_name_usecase/renaming_set_name/   shows a kept name on the roster at once
    usecases/forget_account_usecase/  — on auth.v1.AccountDeleted: the profile, the guest code and the renames, then player.v1.PlayerForgotten
    usecases/rename_player_usecase/  — the operator's name for an account; player.v1.AdminService/RenamePlayer; recorded in renames; then player.v1.PlayerNamed
      rename_player_usecase/audit_rename_player/   logs every rename at Warn
    usecases/name_account_usecase/  — a username for an account signed in to with none; on auth.v1.SignedIn; then player.v1.PlayerNamed and PlayerCreated
      name_account_usecase/renaming_name_account/   shows the name on the roster at once
    usecases/tell_sign_in_usecase/  — on auth.v1.SignedIn: player.v1.PlayerSignedIn, every time
  renames/                          Rename (NewRename, TakeSnapshot), Snapshot, Operator, Reason (ReasonOf, ErrInvalidReason): every
                                    username an operator gave, with the old one; the Storage port, which only writes
    postgres_rename_store/          the Storage over player.renames
    inmemory_rename_store/          the same port in a slice, and Kept for a test, behind the testing tag
    usecases/prune_renames_usecase/ deletes the renames past player.renameRetention; Runner, log_prune
  departures/                       Departure (Decided, Leaving), Decision (Leaves, Returns, TakeSnapshot): an account its player
                                    asked to delete, hidden from the others; the Store port and its contract suite
    postgres_departure_store/       the Store over player.departures, under an advisory lock per account
    inmemory_departure_store/       the same port in maps, behind the testing tag
    usecases/decide_departure_usecase/  — the latest decision kept, then player.v1.PlayerLeaving or PlayerReturned
    usecases/forget_departure_usecase/
  presence/                         Visit, Entry, RosterOf, TTL: who is playing
    inmemory_visit_storage/         the last visit of each account, capped, keyed, pruned every 5s (a Runner), and each change to its subscribers
    usecases/announce_usecase/  listen_for_events_usecase/  move_visit_usecase/  forget_visit_usecase/
  playerv1controller/               PlayerService, InternalService and AdminService (bags), the session interceptor
    get_profile_handler/  set_name_handler/  set_color_handler/  introduce_player_handler/  get_player_handler/
    get_players_handler/  get_nameplates_handler/  get_accounts_handler/  announce_handler/  leave_handler/  get_roster_handler/  listen_for_events_handler/
    admin_get_player_handler/  search_players_handler/  rename_player_handler/  list_renames_handler/   AdminService's
                                    own; it serves get_players_handler too
      admin_get_player_handler/admin_player_query/   PostgresQuery: who one account is, a guest's too
      search_players_handler/matches_query/   PostgresQuery: the accounts a part of a name finds, straight from SQL
      list_renames_handler/renames_query/   PostgresQuery: the operator's renames, newest first, one account's or everybody's
    get_profile_handler/profile_query/  get_player_handler/player_query/  get_players_handler/players_query/
                                    PostgresQuery: the response straight from SQL
    get_roster_handler/roster_query/    MemoryQuery: the roster, from its Lines port
      inmemory_roster/              Lines from the visits in memory (presence.RosterOf)
    inprocess_rename_rule/          GetProfile's Renames port: when a name may change again, from players.NameChoice
    playerread/                     what the queries share and that knows only the wire: KeptColor
    caller/                         the account on the context, or Unauthenticated
    playermessage/                  Profile and the roster lines as player.v1 messages, for the commands and the adapters
  subscribers/                      the edge for events, as the controller is for the wire
    account_deleted_subscriber/  signed_in_subscriber/  signed_out_subscriber/  signed_in_account_subscriber/  signed_in_told_subscriber/
    deletion_requested_subscriber/  restored_subscriber/  deletion_requested_account_subscriber/
    log_subscriber/
  migrations/
```

- **The caller is the account in the click token.** Every call of `PlayerService` but `GetRoster`, `GetPlayer`, `GetNameplates` and `ListenForEvents` sits behind `cpconnect.NewSessionInterceptor`, always enforcing, with the process's one verifier (`props.Sessions`), on the key `auth` hands over the internal listener, as every module does. No token, a bad one, or a token with no account (the deprecated mint) is `Unauthenticated`. `player_session_checks{verdict}` counts the verdicts.
- **`GetProfile`** answers the account id and its name, empty when none was chosen, when the name may change again, and its color. The caller's own stats, titles and fronts are progression's `GetStats`, `GetTitles` and `GetFronts`.
- **`SetName` chooses a username.** `players.NameOf` is the rule. It puts the name in **NFC** and cuts the spaces (U+0020) at its ends — the same text, as it shows — and changes nothing else. Then:
  - **3 to 15 characters, counted in code points after NFC**, as postgres' `char_length` counts, so both agree. A letter with a combining mark NFC cannot compose counts two.
  - **Each is a letter of any script (`\p{L}`), a combining mark (`Mn`, `Mc`) right after a letter or a mark, at most 3 in a row, a decimal digit (`Nd`), `_` or a space.** Never two spaces in a row. So emojis, punctuation, symbols, controls, enclosing marks, and every other space (NBSP, ideographic) are refused. **Invisible characters are refused by name**, since some are letters or marks: Hangul fillers (U+3164, U+115F…, `Other_Default_Ignorable_Code_Point`) and variation selectors; zero-width, bidi and soft hyphen are format characters, refused as not letters. **A letter of a historic script is refused** (`cpscripts.Historic`, see [Shared](shared.md)): cuneiform, hieroglyphs, runes and the like, which players used in names and chat spam (Nov 2026).
  - **A slur or a Nazi reference is refused however it is spelled, anywhere in the name** (`players.DefaultBannedNames`,
    matched by `shared/cpphrases` with `Anywhere`, since a name is one word): `Hitler88`, `xXHitlerXx`, `H1tl3r`,
    `N1gg3r`, and `Snigger`, the slur's usual disguise in a name. The chat lets "Hitler" alone pass in a sentence; a
    name has no conversation, so it is listed here. Nothing that is a plain word inside other words (`nazi` is in
    `Nazir`) nor a number alone (`88`): what the list misses, the daily audit renames (see
    [Chat moderation](../../../docs/moderation.md)). The refusal is `ErrInvalidName` wrapping `ErrBannedName`, which
    never says the phrase. **Only a name being given is checked** (`NameOf`): a name kept before stays, and is the
    audit's. A name suggested at a first sign-in that is refused gets a drawn one, and `random_name_generator`'s test
    reads all 144,000 names it can draw, every number included, since a digit reads as a letter.
  - **One script** (`oneScript`): the letters' scripts, Common and Inherited left out, are one, or a mix UTS #39 calls *highly restrictive* — Latin with Han, Hiragana and Katakana; with Han and Bopomofo; or with Han and Hangul. So `Adа` (Cyrillic `а`) and `guеst_x` are refused.
  - **Not starting with `guest_` once folded** (`Name.Folded`), which the chat puts before every guest's name. Folded, so `ＧＵＥＳＴ_x` and `gueſt_x` are refused too.

  A name that breaks a rule is `InvalidArgument` (`ErrInvalidName`). The name keeps the case it was typed in.
- **A player changes its name at most once every 7 days**, so a name chosen to pass for another player has to be kept. It was 30 days until 2026-10-25. `Profile.Renamed` applies `players.NameChoice`: the time the player last chose its name, plus 7 days, is the earliest next choice. A change sooner, by a nanosecond too, is `players.RenameRefusal` (`ErrRenamedTooSoon`), which `SetName` answers as `ResourceExhausted` with a `player.v1.RenameRefusal` detail: `renamable_at_unix_ms`. The use case checks it after the name and the link, and before the save.
  - **The first name a player picks is free**: an account with no profile (`players.Unnamed`) and a name given at sign-in (`CreateProfile`, which keeps no choice time) start no wait. Every `SetName` that is kept starts it again, the same name in another case too; a refused one (invalid, taken, too soon) does not.
  - **The time is `profiles.name_chosen_at`** (migration `20261015120000_name_chosen_at`), NULL until the player chooses. The migration gave the names kept before it no time, so each player's next change is free. `SaveProfile` writes it with the name.
  - **`Profile.renamable_at_unix_ms`** (0 when the name may change now) is on `SetName`'s answer and on `GetProfile`'s. `profile_query` reads the time in SQL and asks the rule through its `Renames` port (`inprocess_rename_rule`), so the 7 days are only in `NameChoice`.
  - Two `SetName` calls at the same moment can both pass the rule: the last one saved wins, and the wait starts from it.
- **`SetColor` picks the color a username is drawn in**, from `player.v1.NameColor`: the proto enum is the list,
  as `chat.v1.Reaction` is for reactions, and the number is what is stored, so a value is never renumbered or
  reused. `playermessage.ColorOf` refuses a number the proto does not name (`InvalidArgument`, `ErrInvalidColor`);
  `NAME_COLOR_UNSPECIFIED` is accepted too, and the client draws it grey, as a guest; it no longer offers it. It is one
  column, `profiles.color` (migration `20261002120000_name_colors`, `smallint`, 0 by default), written by
  `Store.SaveColor` alone: `SaveProfile` never writes it, so a rename keeps it. **Only a username has a color**:
  an account with no profile row is `FailedPrecondition` (`ErrNoProfile`), and a guest is drawn grey by the client
  whatever it holds. The roster line follows on the client's next announce, which it sends a second after a pick.
- **Known risk: whole-script confusables.** The one-script rule stops a name that swaps one letter for a lookalike of another script, not a name written wholly in a script that looks like another: `Аԁа`, all Cyrillic, is not `Ada`, and both can be held. Closing it needs the UTS #39 skeletons (`confusables.txt`, vendored by a make target) folded into the unique key. Nothing on screen tells two such names apart today: the chat shows no tag beside a name any more.
- **Usernames are unique ignoring case** (`Name.Folded`): **NFKC_Casefold**, the NFKC of the Unicode full case fold of the NFKD, so `Straße` is `STRASSE`, `Émile` is `éMILE`, and a full-width `Ａｄａ` is `ada`. A name another account holds is `AlreadyExists` (`ErrNameTaken`); an account setting its own name again, in any case, is not refused, and a rename or a deleted account frees the old one. **The game computes the fold and the store keeps it** in `profiles.name_folded`: postgres' `lower()` depends on the database's locale (a C locale folds ASCII only) and folds neither `ß` nor full-width letters, so an index on it would disagree with Go on non-ASCII names. Postgres holds uniqueness, not a read before the write: a unique index on `name_folded`, whose violation (`23505` on `profiles_name_key`) `postgres_player_store.SaveProfile` answers as `ErrNameTaken`, so two players asking for one name at once cannot both get it. The in-memory store compares `Folded` under its lock, and `StoreContractSuite` pins both.
- **The table's `CHECK` holds what it can say without the locale**: 3 to 15 `char_length`, `IS NFC NORMALIZED` (a UTF8 database), no space at either end or two in a row, no ASCII but letters, digits, `_` and space, no C0 or C1 control, and none of the zero-width, bidi and other invisible format characters (a range list in the migration); and `name_folded` not empty and not starting with `guest_`. Letter categories and scripts are Go's alone: postgres' regex classes depend on the locale too.
- **Only a linked account may hold one.** `set_name_usecase` checks the name first, so a name no account may hold costs no call, then asks its `Accounts` port whether the caller signed in with a provider: `rpc_account_reader` calls `auth.v1.InternalService/GetAccount` over the internal listener on every `SetName` (2s timeout), since an account links at any time and a name is chosen rarely. A guest is `PermissionDenied` (`ErrNotLinked`). Auth answers `linked` false for an unknown account or an id that is not one; **a failure to ask is a real error**, the error net's `internal`, never a guest. With `auth` off that call 404s, so no name can be set.
- **A signed-in account is given a username** when it has none, so every player who signs in has one. On `auth.v1.SignedIn` (`player-names-sign-ins`), `name_account_usecase` asks `players.GeneratedNames.Assign`, which saves with `Store.CreateProfile`: an insert that refuses an account with a profile (`ErrProfileExists`), so a name chosen at the same moment is never replaced.
  - **The identity's name comes first** (`SignedIn.identity_name`, CrazyGames' username), so friends know each other there. CrazyGames allows `.` and 20 characters: `suggestedNameOf` turns each `.` into `_` and cuts to 15 code points (`Nova.42` is `Nova_42`), then `NameOf` decides. When it refuses the result (`guest.nova`) or another account holds it, a name is drawn. It is a name given at sign-in, so the first `SetName` is free.
  - `random_name_generator` draws an adjective, an animal and two digits (`BraveFox42`; 40 × 40 × 90 names, every one valid under `NameOf`, which its tests pin), and a name another account holds is drawn again, 10 draws at most as for guest codes (`ErrNoFreeName`, logged; the next sign-in tries again). Postgres decides what is taken: the unique index on `name_folded` refuses the second of two accounts saving one name at once, whatever both read before. `renaming_name_account` renames the roster line at once. The player changes it with `SetName` as any name. A guest is never named: it has not signed in.
- **An account with a profile is a player, and player says when one is made: `player.v1.PlayerCreated`**, through player's outbox (migration `20261030130000_outbox`, relay `player-outbox`), in the transaction that makes the profile. Two use cases make one: `name_account_usecase`, when `Assign` answers that it made the profile (an account that had one makes nothing), and `set_name_usecase`, when the account had no profile yet. Planet grants a new player a refill ([A guest's first sign-in](sign-in-refill.md)). **`CreateProfile` raises no error in that transaction**: it inserts with `ON CONFLICT DO NOTHING`, then reads whether the account has a profile to tell `ErrProfileExists` from `ErrNameTaken`. A unique violation would end the transaction, and the next draw could not run in it. `TestATakenNameLeavesTheTransactionToDrawAgain` pins it.
- **The operator reads players on `player.v1.AdminService`**, for the admin page's Players and its home (see [Operator
  tools](admin.md)). All four are reads, and none logs or writes anything:
  - **`GetRoster` is the public service's, handler and all**: who is playing now. The admin page calls the admin
    listener alone, so the public route is out of its reach.
  - **`GetPlayers(account_ids)` is the internal service's, handler and all**: the same `get_players_handler`
    serves both, so the admin page names a page of players exactly as the chat and the season board do, and a
    read never draws a guest code. The planet's `TopPlayers` names no row, so the page asks both, and joins them. Planet
    could not ask `player` when this was built: `player` asked planet for the fronts, and a cycle is the one thing two
    modules may not have. Progression asks for the fronts since 2026-11-04.
  - **`GetPlayer(account_id)` is who one account is**, a guest's too, which the public `GetPlayer` refuses
    (`admin_get_player_handler`, `admin_player_query`): its name (the username, `guest_` and the code, or empty for
    an account never shown), `guest` (no username), `linked` and `created_at_unix_ms` from auth (`GetAccount`, one
    call), `admin` and its color. How far it went (its stats, messages, titles and fronts) is
    `progression.v1.AdminService/GetProgress`, which the back office asks beside it (see [Progression](progression.md)).
    `NotFound` (`admin_player_query.ErrNoPlayer`) is only an account with no name, no guest code, and that auth knows
    nothing of (not linked, no creation date): an account never shown that auth made answers with an empty name. Until
    2026-11-04 the stats told an account never shown from one that never was; player keeps no stats now, so auth tells
    it. Fields 7 to 12 are reserved. An id that is not an account is `InvalidArgument`; a failure to ask auth is the
    error net's `internal`. `no-store`.
  - **`SearchPlayers(name, limit)` finds an account by a part of its name**, for the player the operator only knows
    from the chat: the ledger lists who played in its 72 hours, and a banned player has played nothing since. The
    part is folded as a username is (`players.FoldedOf`, the same NFKC_Casefold), so `eire` finds `EIRE`, and
    matched anywhere in `profiles.name_folded`, or in `guest_` and the code for an account with no username, as
    `GetPlayers` names it: a code an account went by before it chose a name finds nothing. The same name first,
    then the names it starts, then the shortest, then by name. `limit` is 20 when 0, 500 at most, and `total`
    counts every account found. Each is a `PlayerMatch`: its account id, its name and `guest`, all a row of the
    search shows. `matches_query` reads them in one statement (a scan of both tables: no index serves a match
    inside a name), and its test pins that each is named as `GetPlayers` names it. An empty part is
    `InvalidArgument` (`ErrNoName`). `no-store`.
  - **It was `FindAuthors` for a day** (#450), answering `Author`s through `authors_query`: the name followed the
    chat's type it reused, and "author" means nothing to an operator browsing players. Only the back office called
    it, so it was renamed in one step.
- **The operator changes a username with `player.v1.AdminService/RenamePlayer(account_id, name, reason)`**, for a player stuck
  with a bad name for 7 days (`rename_player_usecase`, wrapped in `audit_rename_player`, which logs every call at
  Warn with the account and the name typed).
  - **`SetName`'s rules hold but the wait**: `NameOf`, then unique ignoring case. An invalid name is
    `InvalidArgument`, a name another account holds `AlreadyExists`. The answer is the name as kept.
  - **Only an account with a username** (a profile): a guest or an account never named is `NotFound`
    (`ErrNoProfile`). It makes no profile, so it makes no player.
  - **It is not the player's choice**: `Profile.RenamedByOperator` clears `name_chosen_at`, as a name given at
    sign-in keeps none, so the player's next `SetName` goes at once.
  - **Nothing pushes it in the game**: the chat names a sender when it shows a message, and the roster takes the name
    at the player's next announce, within 30s. It tells `PlayerNamed`, in the transaction of the save, so the Discord
    member's nickname follows (see [Discord pings](alerts.md#the-nickname-linksnamer)).
  - **It takes a `reason`, and is kept** (`renames`, `player.renames`, migration `20261106120000_renames`): the
    account, when, the name it held before and the new one, the reason, and who made it (`operator_id` and `client`,
    as the admin gate admitted the call: see [The moderation history](chat.md#the-moderation-history)), in the
    transaction of the save. The reason is `renames.ReasonOf`, the chat's rule (optional, 500 characters at most);
    one it refuses is `InvalidArgument` and renames nothing. `audit_rename_player` logs it too.
- **`player.v1.AdminService/ListRenames(account_id, since, limit)`** reads them (`renames_query`): newest first, one
  account's or every account's (an id that is not a UUID is `InvalidArgument`), made at `since` or after, 50 unless
  asked, 500 at most; `NO_SIDE_EFFECTS`, a read-only tool of the connector, and `no-store`. It is the names' half of the
  moderation history; the chat's mutes and removals are `chat.v1.AdminService/ListModeration`, and why each context keeps
  its own is in [Chat](chat.md#who-owns-which-history).
  - **Kept for `player.renameRetention`** (90 days), and deleted past it by `prune_renames_usecase`, at boot and every
    `player.pruneInterval` (1h). **Forgotten with the account**: `forget_account_usecase` deletes the account's renames
    and takes it off the ones it made as an operator, in the transaction that deletes its profile.
- **Migration `20260918190000_unicode_usernames`** moves to this rule. Every name kept before is ASCII of 3 to 20, so only length breaks the new rule: **a name over 15 is cut to its first 15 characters**, the player keeping most of it. A cut name that another would then hold, ignoring case — a name of 15 or fewer, or an older cut name (`updated_at`, then `account_id`) — is **deleted** instead, and its player chooses again; an untouched name is never the one to go. Then it fills `name_folded` with an ASCII-only `translate()` (exact for these names, in any locale), replaces the `CHECK`, and moves `profiles_name_key` to `name_folded`. The down migration deletes the names the old rule refuses, drops the column and puts the old `CHECK` and the `lower(name)` index back; cut names stay cut. `migrations_test.go` migrates to the version before, writes names, and migrates the rest.
- **Migration `20260917180000_usernames`** deletes the profiles whose name breaks the new rule, then, of names that differ only in case, every one but the oldest (`updated_at`, then `account_id`), replaces the name `CHECK` with the new pattern and adds the unique index. No client called `SetName` before usernames existed, so nothing a player chose is lost. The down migration drops the index and puts the old `CHECK` back; the deleted rows stay deleted.
- **A guest code names an account that chose no username**: 6 lowercase hex characters (`players.GuestCode`), shown as `guest_` and the code (`players.DisplayNameOf`). It is drawn by `random_code_generator` the first time the account is shown (`players.GuestCodes.Assign`), kept in `player.guest_codes`, and deleted with the account, which frees it. **It is unique**: the table's `guest_codes_code_key` refuses a code another account holds, the store answers `ErrGuestCodeTaken`, and `Assign` draws again, up to 10 times (`ErrNoFreeGuestCode`). An account keeps its first code: a second save for it changes nothing, so two first announces at once leave one code. **It does not change with the network**, unlike the address, and says nothing about the account or the address. A linked account with no username keeps the code it had as a guest until it picks a name.
- **Every read of a player answers one structure, `player.v1.Player`**: its `account_id`, the name the game shows for it (the username, or `guest_` and the code), `admin`, `color`, and `guest` for an account with no username, whose other fields are unset. `PlayerService/GetPlayer`'s card, `InternalService/IntroducePlayer` and `InternalService/GetPlayers` all answer it. **`created_at_unix_ms` is auth's**, so only `GetPlayer` asks auth for it (`GetAccount`, one call); the other two leave it at 0 rather than ask auth on every message and every board. **It carries no badge**: an account's streak and worn title are progression's `GetBadges` ([Progression](progression.md#the-badges)), which the page reads by the account id beside the name; until 2026-11 the player carried both, copied into every message, roster line and standing.
- **`InternalService/IntroducePlayer(account_id)`** is for the chat, about to show who posts: the player the account is (`players.PlayerOf`), which the chat sends no GIF from when it is a guest. **`introduce_player_usecase` gives a guest its code the first time it is introduced** (`Assign`, then read again), and the roster's announce and move use the same use case, so the chat and the roster always show one name for one account. It is not a pure read, hence its verb. An empty id, or one that is not an account, is `InvalidArgument`: the chat refuses a sender with no account before asking.
- **`InternalService/GetPlayers(account_ids)`** is the same question about a page of accounts, for a module that names them on the server — the chat's operator list naming a page of messages, support naming the accounts of its reports, clips naming a cast, alerts naming a Discord member. What the page draws is `GetNameplates` (below), the same read. Each player also says `guest` for an account with no username, which the standings never rank. **It is a pure read**: unlike `IntroducePlayer` it draws nobody a guest code, so a read path never writes. That is the whole reason the two exist side by side; `IntroducePlayer` belongs where somebody is about to be *made* visible, `GetPlayers` where somebody is merely being shown. One statement (`players_query`: an `unnest` of the ids left-joined onto `profiles` and `guest_codes`) answers the page, in the order asked; the SQL names a guest `guest_` and its code, as `IntroducePlayer` does, and carries no badge either; a repeated id is answered once, and **an account it cannot name — never shown, or deleted — is left out** rather than failing the call, so the caller decides what stands in its place. An id that is not an account is still `InvalidArgument`: a caller holding one has a bug.
- **`PlayerService/GetNameplates(account_ids)` is every name the page draws: the chat, the boards and the rounds**, read
  by the browser by account id beside each name, as it reads the badges from progression (see the frontend's
  [Nameplates](../../frontend/docs/nameplates.md)). Each `Nameplate` is `account_id`, `name`, `color`, `admin` and
  `guest`. **It is `GetPlayers`' own query** (`get_nameplates_handler` over `players_query`), so a name is drawn the
  same wherever it is read, and nothing the other players cannot name leaks: an account unknown, never shown, deleted
  or asked deleted is left out, and the page draws it `[deleted]` where it shows one. It draws no guest code.
  - **At most 100 ids a call** (`get_nameplates_handler.MaxAsked`), a repeated one counted each time; more is
    `InvalidArgument` (`ErrTooMany`) and reads nothing. An id that is not an account is `InvalidArgument`.
  - **It needs no token** (the session interceptor does not list it: names are public), is `NO_SIDE_EFFECTS`, and
    answers `no-store`: the page is the cache, and an answer kept on the way could undo a rename it then learns of.
  - **Why the page reads it** (2026-11-06): the chat and scoring asked `GetPlayers` at every read only to copy the names
    into their answers. A board waited on player, and a change of `Player` touched both. They answer account ids now,
    and their copies (`ChatMessage.author_name`, `author_admin`, `author_color`, `Quote.author_name`, `author_color`,
    `ReactionCount.reactors`, `Standing.name`, `color`) are `reserved`. The chat still introduces a sender
    (`IntroducePlayer`): the guest code and the no-GIF rule are game rules. Scoring keeps its own record of who may be
    ranked, from player's events (see [Scoring](scoring.md#who-may-be-ranked)).
- **They were `GetAuthor` and `GetAuthors` until 2026-11-04**, each answering a message of its own (`GetAuthorResponse`,
  `Author`), and `players.Author` was the domain's: "author" is the chat's word for who posts, and the chat is
  downstream of player, so it had no place in player's language. Most callers were not even the chat. `Author` and
  `Player` said the same thing in one package, so they are one message now, `Player`, with the `account_id` the page
  needed. A hard switch: the internal callers are in the same binary, and the back office's `AdminService/GetPlayers`
  needs the admin page reloaded.
- **The tag is private.** `players.TagOf` (SHA-256 of `player.tagSalt`, a NUL and the address, cut to 6 hex characters) is kept on a roster visit so the storage can cap the visits of one address, and never leaves the server.
- **Who is playing is `presence`, in memory only.** A client calls **`Announce(country_id)`** with its click token when it gets one, when its flag or name changes, and every 30s. A visit counts for `presence.TTL` (90s), so a hidden tab whose timers fire once a minute stays on; the prune runs every 5s. A restart empties it, and clients fill it again within one interval. **`Leave`** takes the caller off at once: a client sends it with `keepalive` when its page closes. **`GetRoster`** needs no token (the session interceptor does not list it), is `NO_SIDE_EFFECTS`, and answers `public, max-age=5`; it stays for clients from before the stream.
  - **`ListenForEvents` is the roster, live.** It needs no token and reads none: nothing on it is anybody's own. The titles an account earns have their own stream since 2026-11-04, progression's ([A title earned reaches the player at once](progression.md#a-title-earned-reaches-the-player-at-once)); `PlayerEvent.title_earned` (5) is reserved. The first event is the whole roster (`roster`, in `GetRoster`'s order); each one after is a line that joined or changed (`entry`), or the key of one that left (`left`); `heartbeat` every `httpServer.streamHeartbeat`. **A line's `key` is the storage's**: a counter handed to an account's first visit, kept by every later announce, `Move` and `Rename`, so a sign-in renames one line rather than adding one. It says nothing about the account; **the line's `account_id` does**, so a client can say which player it means with an id and never with a name. `Storage.Subscribe` answers the roster and registers the subscriber under one lock, so nothing is lost between the two, and every change is published under that lock, in order. An announce that changes nothing on the line (name, flag, guest, admin, color) publishes nothing, unless the visit had gone stale. **A subscriber `SubscriberBuffer` (256) changes behind is closed, not skipped**: a skipped change would leave its roster wrong for good, and the stream ends, reconnects and starts from a whole roster.
  - **One entry per account**, so the tabs of one browser and the devices of one signed-in player are one line. A visitor who never got a click token is not listed: announcing must not cost a Turnstile mint.
  - **Named as the chat names a sender**, by the same use case (`introduce_player_usecase`): the username, or `guest_` and the guest code, with its color, read on every announce. **A line carries no badge**: the page reads its streak and title from progression's `GetBadges` by the line's `account_id`, so a title worn or a streak extended reaches the others within the page's 5 minutes, not on the stream. **No address, and no hash of one, is on a line.**
  - **Sorted**: players with a username first, then guests; each group by name ignoring case, then by key.
  - **Caps, against a script minting accounts** (`inmemory_visit_storage`): at most 10 accounts per tag, where a new account pushes out the tag's oldest visit, and 10,000 in all, where a new account is not recorded. The mint throttle already bounds how fast one address makes accounts.
  - An unknown country is `InvalidArgument`; a failed profile read is the error net's `internal`, and nothing is recorded.
  - **A sign-in, a new name, a sign-out and a deletion change the roster at once, with no announce.** The client drops its click token on each of these, and a new one waits for a click, so waiting for its next announce left a guest line on the roster, or two lines, for up to 90s. So: `auth.v1.SignedIn` moves the browser's visit to the account it is on now, under that account's name (its username, or its own guest code), over any visit the account held (`move_visit_usecase`, `Storage.Move`; one account before and after changes nothing, and reads nothing). `SetName` renames the caller's visit once the name is kept (`renaming_set_name`, `Storage.Rename`). `auth.v1.SignedOut`, `auth.v1.AccountDeletionRequested` (`player-presence-deletions`) and `auth.v1.AccountDeleted` take the account off (`forget_visit_usecase`, `Storage.Forget`); another device still signed in announces again within 30s. An account with no visit is left off by all of them: its browser never announced.
- **An admin of the game is `player.profiles.admin`**, false by default. **The game never sets it**: an operator flips it in the database (below), and `SaveProfile` never writes it, so a rename keeps it. It only means something beside a username: `IntroducePlayer` and `GetPlayers` answer `admin` for the chat, which shows the crown on a message whose account is an admin **now** — it is read when the message is shown, not stamped on it, so a player that stops being an admin stops looking like one on what it already said; `presence.RosterOf` sets `Admin` on a roster entry only when it is not a guest; `GetPlayer` answers it too. The frontend draws a crown on all three. An announce reads the profile, so a new admin shows on the roster within 30s, and in the chat the moment anybody reloads it.

  ```bash
  ssh deploy@YOUR_IP
  cd /opt/clickplanet/deploy/vps
  docker compose exec postgres psql -U clickplanet -c \
    "SELECT account_id, name FROM player.profiles WHERE name = 'TheUsername'"
  docker compose exec postgres psql -U clickplanet -c \
    "UPDATE player.profiles SET admin = true WHERE account_id = '<account_id>'"
  ```

  **Two steps, and the write names the account.** A name is a handle: a rename frees it, and another account can take it between the moment you read it and the moment you write. So read the id, check it is the player you mean, and act on the id. An account with no username has no profile row, so it cannot be one: pick the name first.
- **`GetPlayer(account_id)` is what anybody may know about who a player with a username is**: the name it holds now, its color, `admin`, and `created_at_unix_ms`, when auth made the account (as a guest or by a first sign-in, so a guest who signs in keeps its first day). How far it went (its stats as of today, its titles, the countries it plays for and against) is progression's `GetProgress`, which the page asks beside it ([The public card](progression.md#the-public-card)); fields 2 and 6 to 9 are reserved. It needs no token (the session interceptor does not list it), is `NO_SIDE_EFFECTS`, and answers `public, max-age=10`. **The account id is the only thing that names a player**: a chat message, a roster line and a standing each carry it, so a card opened from one is the card of that account, whatever it is called now. A name cannot do that: a rename frees it, and the next account to take it would get the first one's card. **It never answers the account id**: the caller holds it. An id that is not an account, an empty one included, is `InvalidArgument` (`players.AccountIDOf`). **A guest is its name alone**: `guest_` and its code, and `guest`, with every other field unset, and nothing asked of auth. It shows no streak and wears no title anywhere else, and the card is what a username gives; the globe names who holds a tile with it (planet's `GetTileHolder`). An account the game never named is `NotFound` (`player_query.ErrNoPlayer`): one that never existed, a deleted one, or one never shown, which has no guest code. `rpc_account_reader.CreatedAt` asks `auth.v1.InternalService/GetAccount` on each call, which also answers `created_at_unix_ms` (zero for an account auth does not know, and the answer then carries zero). **A failure to ask auth is a real error**, the error net's `internal`, as for `SetName`.
- **`auth.v1.AccountDeleted` deletes the profile, the guest code and the renames** (`player-profiles-accounts`, `forget_account_usecase`), and tells `player.v1.PlayerForgotten` (below). The group was `player-accounts` until 2026-11-04: see [The deploy that moved the others to player's events](#the-deploy-that-moved-the-others-to-players-events).
- **Errors are real errors.** Every `Store` method returns one; absence is `ErrNoProfile`, which the use cases answer as an empty profile. On a request a database error is the error net's `internal`. On an event it is counted in `events_failed_total`, logged by `log_subscriber`, and the event tried again, then buried in `events:dead`. Each event's write has a 5s timeout (`subscribers.Timeout`), so a stuck database cannot hold a subscriber.
- **`players.StoreContractSuite`** is the port's behaviour, run on `inmemory_player_store` and on postgres. The use cases are tested over the in-memory one, which can `FailWith` an error.
- **The pool closes after the subscribers stop** (`cppg.CloseAfter`), so what they read before the shutdown is still written.
- **No foreign key to `auth.accounts`**: the schemas are each module's own, and the event is how a deletion crosses.

## What player tells the others

**Player is the other modules' one source on an account**, so they need not hear auth: auth is player's detail. Each
event goes through player's outbox (relay `player-outbox`; `outbox_event` gained `PlayerLeaving`, `PlayerReturned` and
`PlayerForgotten` in migration `20261104190000_departure_events`, `PlayerSignedIn` in
`20261104220000_signed_in_event`, `PlayerNamed` in `20261105120000_named_event`), in the transaction of what it tells, since a lost one would leave a module
disagreeing with player for good (see [What may not be lost](composite-layer.md#what-may-not-be-lost-the-outbox)).
**Who hears them**: `PlayerCreated`, planet (the sign-in refill), progression (a nameplate) and scoring (a player it
may rank); `PlayerSignedIn`, missions (what a guest finished that day is paid); `PlayerNamed`, alerts (the Discord
member's nickname); `PlayerLeaving` and `PlayerReturned`, progression, scoring (a player it ranks no more, then again)
and alerts (a nickname taken off and given back);
`PlayerForgotten`, every module that keeps something of an account: progression, planet, chat, scoring, support,
missions, alerts and clips. `GetAccounts` is asked by progression and missions. Since 2026-11-04 no game module but
alerts hears or asks auth, which needs it for a Discord join's token (see [Discord pings](alerts.md)):
`TestOnlyPlayerAndAlertsSpeakAuthsLanguage` pins it.

- **`player.v1.PlayerCreated{account_id}`**: an account became a player (above).
- **`player.v1.PlayerSignedIn{account_id}`**: a player signed in to the account, which is signed in now.
  `tell_sign_in_usecase` records it on **every** `auth.v1.SignedIn` (`player-sign-ins`, `signed_in_told_subscriber`):
  a first link, a sign-in to an account linked long ago, a switch from the browser's guest to another account, a new
  account. It is not `PlayerCreated`, which is told once per profile, never for an account that held one already, and
  not when no name could be drawn (`ErrNoFreeName`); its own group, so a name drawn or not changes nothing to it.
  Missions pays then what the account finished as a guest (see [Missions](missions.md)).
  `TestEverySignInIsToldToTheOtherModules` pins it; migration `20261104220000_signed_in_event` gives it its place in
  `outbox_event`, and `TestEveryEventPlayerTellsHasAPlaceInItsOutbox` pins every event's.
- **`player.v1.PlayerNamed{account_id}`**: the account's username was set. `name_account_usecase` records it with
  `PlayerCreated` when it draws the first name, `set_name_usecase` with every name it keeps (a first one, a rename, the
  same name in another case), and `rename_player_usecase` with every operator's rename, each in the transaction of the
  save; a name refused (invalid, taken, too soon) tells nothing. **The name is not in it**: the alerts module asks
  `GetPlayers` for the name now, so two told close together, or one told twice, leave it with the latest, whichever
  stream it read first. `TestEveryNameKeptIsTold`, `TestARenameIsTold` and the name-account test pin it.
- **`player.v1.PlayerLeaving{account_id, decided_at}`** and **`PlayerReturned{account_id, decided_at}`**: its player
  asked to delete it, or a sign-in brought it back (see [An account asked deleted](#an-account-asked-deleted)).
  `decide_departure_usecase` records one **for every decision, applied or not**, in the transaction of the departure:
  the two come on two streams, so a listener keeps the latest by `decided_at`, as player does, and needs every one to
  do so. `TestEachDecisionIsToldToTheOtherModules` pins it.
- **`player.v1.PlayerForgotten{account_id}`**: player forgot the account, its name and its guest code, and what
  another module keeps of it goes too. `forget_account_usecase` records it on `auth.v1.AccountDeleted`
  (`player-profiles-accounts`), in the transaction that deletes the profile and the guest code, **for every account deleted,
  even one player kept nothing about**: a guest that took tiles, chatted or did a mission has no profile, and the
  modules that hear it keep its takes, its stats and its seen mark. The roster and the
  departure forget the account on their own groups (`player-presence-accounts`, `player-departures-accounts`).
  `TestAnAccountPlayerNeverKnewIsToldForgottenToo` pins it.
- **`InternalService/GetAccounts(account_ids)`** is the read beside them: what auth says of a page of accounts, as
  player's projection of `auth.v1.InternalService/GetAccounts` (`get_accounts_handler`, through
  `rpc_account_reader`): each one's `account_id`, `linked` and `created_at_unix_ms`, in the order asked, once each.
  An account auth does not answer is left out, and the caller reads it as a guest with no date. An id that is not an
  account is `InvalidArgument`, and reads nothing; a failure to ask auth is the error net's `internal`.
  `NO_SIDE_EFFECTS`. Progression asks it on each title award and once per page of a reconciliation, missions whether
  an account is signed in before it pays.

## An account asked deleted

**Auth keeps an account its player asked to delete for 30 days, and to every other player it is deleted at once.**
See [Auth](auth.md#signing-out-deleting-pruning). `player` is where that is decided, since every name another player
reads comes from here: `GetNameplates` (every name the page draws: the chat, the boards and the rounds), `GetPlayers`
(the clips' casts, the operator's lists), `GetPlayer` (the card on a tile, a territory, a standing or a chat line) and
the roster. It tells each decision (`PlayerLeaving`, `PlayerReturned`), so progression hides the account's badges and
progress too (see [Progression](progression.md#who-the-others-see-nameplates-and-departures)), and scoring ranks it no
more (see [Scoring](scoring.md#who-may-be-ranked)).

- **`departures`** keeps, per account, whether it is leaving and when that was decided (`player.departures`, migration
  `20261102150000_departures`). `auth.v1.AccountDeletionRequested` decides it leaves
  (`player-departures-requests`, `departures.Leaves(requested_at)`), and `auth.v1.AccountRestored` that it returns
  (`player-departures-restores`, `departures.Returns(restored_at)`), both through `decide_departure_usecase`, which
  tells the decision to the other modules in the same transaction.
- **The latest decision wins** (`Departure.Decided`): the two events come on two streams, so a restore can be heard
  before the request it undoes, and either can come twice. A decision with no time is refused (`ErrNoTime`), so a
  broken event is buried rather than read as the oldest one.
- **`GetNameplates`, `GetPlayers` and `GetPlayer` leave a leaving account out** (`NOT EXISTS` on `departures` in
  `players_query` and `player_query`), exactly as they leave out a deleted one: the readers already show a player
  nobody can name as deleted ("[deleted]" in the chat, "This player is gone." on a card, no row on a board, which
  scoring decides from the same events). Nothing is deleted, so a
  restore shows it again as it was. `IntroducePlayer` (a post's own author), `GetProfile` (the caller's own)
  and the operator's `AdminService/GetPlayer` and `SearchPlayers` still read it; `AdminService/GetPlayers` is the same
  handler as the internal one, so the operator's lists of messages and reports show it unnamed meanwhile.
- **The roster drops it** at the request (`player-presence-deletions`, `forget_visit_usecase`): its sessions are gone,
  so it would only age out.
- **`auth.v1.AccountDeleted` forgets the departure** (`player-departures-accounts`, `forget_departure_usecase`), with the
  rest of the account.

## The deploy that moved the others to player's events

**On 2026-11-04 the modules that heard `auth.v1.AccountDeleted` to forget an account moved to `PlayerForgotten`, and
missions' sign-in to `PlayerSignedIn`**, each on a new group. A new group reads its stream from its start, but the
start is only what is left of it: every subscription trims its stream to the oldest entry one of the stream's groups
has not handled (see [Telling other modules what
happened](composite-layer.md#telling-other-modules-what-happened)). So a `PlayerForgotten` progression's six groups had
handled before the deploy may be gone. An `AccountDeleted` that a module's old group had not handled when the old
server stopped (read at the last moment, or failing and tried again) while player had already told it forgotten would
then never reach that module.

**So player read auth's two streams again, from what is left of them, under new groups.** Its forget moved from
`player-accounts` to `player-profiles-accounts`, and the sign-ins it tells are a new group, `player-sign-ins`. The old
groups nobody subscribes with any more stay in Redis and hold their stream's trim back to the oldest entry each had not
handled, so **every deletion and every sign-in a module had not handled is still in auth's streams at the boot**.
Player tells each one again, from its outbox, after the module's new group exists: every group is made while the
modules are built, before any runner reads or trims. A forget deletes what is left, which is nothing the second time,
and missions pays only what is due, so the ones told twice change nothing. The cost is the entries still in the two
streams told once more: the last few, and whatever an old group had left.

**By hand, once the release is out and the new groups caught up** (`events_lag` 0 for `player-profiles-accounts` and
`player-sign-ins`), not before, since the old groups are what keep those entries:

```bash
XGROUP DESTROY events:auth.v1.AccountDeleted player-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted planet-ledger-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted chat-seen
XGROUP DESTROY events:auth.v1.AccountDeleted seasons-standings-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted seasons-rounds-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted support-notes-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted support-reports-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted missions-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted alerts-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted clips-casts-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted clips-appearances-accounts
XGROUP DESTROY events:auth.v1.AccountDeleted clips-optouts-accounts
XGROUP DESTROY events:auth.v1.SignedIn missions-sign-ins
```

`alerts-accounts` exists only where alerts was ever on. Until they go, each holds its stream back, bounded by
`redis.maxLen`.
