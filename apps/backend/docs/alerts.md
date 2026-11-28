# Discord pings (`internal/alerts/`)

**A player connects Discord once, by signing in with it, and the game's Discord server pings it.** The sign-in adds the
player to the server with the role of the flag it plays for, a sold flag's too, and its username in the game as its
nickname there. The
server pings a flag's role when that flag is attacked, and everybody in the server (`@everyone`) when a battle or a
season starts. Off by default
(`alerts.enabled`); off, the module is not built, `alerts.v1` 404s, which the client reads as no offer, and a Discord
sign-in joins no server.

Why: a newcomer who plays once never comes back on its own, and the players who come back on day 2 to 7 have no way to
hear that their land is attacked. They have Discord open on their phone; a website's notifications they do not allow.

**One Discord connection, not two** (the owner, 2026-11-02). The first version had its own Discord trip, "Get pinged",
beside "Sign in with Discord": two consents on the same application, and nobody could say why both were needed. Now
the alerts module starts no trip of its own: `auth`'s Discord sign-in asks the leave to join the server too, keeps it
for ten minutes, and tells it (`auth.v1.DiscordJoinGranted`); alerts hears it and asks auth for the token (see
[Auth](auth.md)).

## The link (`links/`)

**Made when alerts hears `auth.v1.DiscordJoinGranted`** (`alerts-joins`), which `auth` records once a Discord sign-in
or link is saved, in its transaction, when `auth.discord.joinServer` is on (Discord's scope `guilds.join`). The event
names a grant and never holds the token: `links/rpc_auth_grants` asks `auth.v1.InternalService/GetDiscordGrant` for
the account, the Discord user and the token, on the internal listener. Auth called alerts before (`JoinServer`, which
waited up to 5 s inside `CompleteSignIn`): the most upstream module named a leaf, and auth and alerts called each
other. Now auth names no module, and the join lands a second or two after the sign-in comes back.

- **`join_server_usecase`** asks planet the flag the account pledged (`planet.v1.InternalService/GetAllegianceOf`),
  makes its role if needed, adds the user to the server with it (`PUT /guilds/{guild}/members/{user}`, or the role
  granted when the user is in it already), and keeps the link. An account that never pledged joins with no role: its
  first pledge gives it. The game pledges the page's flag before the trip, so a guest that has not clicked yet
  pledged one.
- **An account linked to this user already changes nothing**: every Discord sign-in grants a join, and a user who left
  the server since is not added back by each one. A grant heard twice is the same: the second finds the link.
- **One Discord user, one account.** A user linked from another account moves here: the first link is forgotten.
- **Alerts never keeps the access token**: it joins the user once. Auth keeps it, sealed, ten minutes, and the bus
  tries a join again for about four. A link keeps the account, the Discord user id, the flag of its role, the name the
  bot gave the user as its nickname (`links.name`, migration `20261105120000_link_names`), and when.
- **Joined, the user is given its username** (below): `discord_join_granted_subscriber` asks `name_member_usecase`
  once the join passed. A nickname that fails may pass is tried again with the join, which finds the link and joins
  nobody twice; one Discord refused is logged, and the user stays linked under its own Discord name.
- **A failure never fails the sign-in**: the sign-in was saved before alerts heard of it. What cannot pass is logged
  and not tried again: a grant auth keeps no more (`NotFound`, `links.ErrNoGrant`), and a token Discord refused
  (`discord.ErrRefused`, a 4xx: no `guilds.join`). The player is signed in, not linked, and the game offers to join
  again. What may pass (a 429 or a 5xx from Discord, auth or planet unreachable) is an error, which the bus tries again.
  The token is in no log line and no error.
- **The role follows the flag pledged.** `alerts-allegiances` hears `planet.v1.AllegiancePledged`, told when a pledge
  sets or changes the flag (see [Allegiance](allegiance.md)): a linked account is granted the new flag's role and loses
  the old one. Before, the role followed the flag of the next tile taken, so a player who changed flag and did not
  click kept being pinged for the old one. A user who left the server (`10007 Unknown Member`) is forgotten: it asked
  for no more pings.
- **A deleted account is forgotten** (`alerts-forgotten`, `player.v1.PlayerForgotten`, which player tells for every
  account auth deletes). The Discord user stays in the server with its roles: that is its own Discord account. The
  nickname the bot gave it is the game's, the username player forgot, so it is taken off first, and Discord shows the
  user's own name again. A user who left the server, or one the bot may no longer rename, is forgotten all the same;
  Discord busy or down keeps the link until the bus tries again. Until
  2026-11-04 it heard `auth.v1.AccountDeleted` (`alerts-accounts`); why the new group misses no deletion is in
  [Player](player.md#the-deploy-that-moved-the-others-to-players-events).
- **Alerts is the one game module besides player that talks to auth, on purpose.** Every other module asks player
  who an account is. Alerts hears `auth.v1.DiscordJoinGranted` and asks auth `GetDiscordGrant` and `GetDiscordUsers`,
  because a join needs the user's OAuth access token, a secret auth keeps sealed and never puts on Redis. Through
  player, player would hold OAuth secrets that are nothing to it: who a player is needs no token.

`GetDiscordLink` answers `offered` (true: a server that answers at all offers it), `linked`, and the server's page
(`#alerts`). It reads `links` straight (a query), with the caller's click token. The public route,
`/alerts.v1.AlertsService/*`, is listed in `deploy/vps/caddy/Caddyfile`: without it Caddy answers 404, and the game
shows no offer.

### The operator (`AdminService`)

On the console and admin listeners, as every operator tool (see [Operator tools](admin.md)):

- **`GetPlayerLink(account_id)`**: the account's link (the Discord user, its role's flag, when), or none.
- **`UnlinkPlayer(account_id)`**: forgets the link and takes the flag's role and the nickname the bot gave off the user
  (`unlink_player_usecase`). The user stays in the server; a user who left it already is unlinked all the same.
  `unlinked` false: there was no link. A role or a nickname Discord would not take off keeps the link
  (`FailedPrecondition`). Unlinked, the game offers that account
  to join again, and its next Discord sign-in joins it again: this is how the offer is tried. Players cannot unlink
  themselves; leaving the server is how they stop the pings.
- **`LinkMembers()`**: links the players already in the server who have no link (`link_members_usecase`), moves every
  linked one to the role of the flag it plays for when it holds another, gives every linked one its username as its
  nickname, and counts them. See below.

### The players a sign-in never joined (`LinkMembers`)

**A Discord account with no link is in one of two places.** The players who signed in with Discord before the
sign-in asked `guilds.join` (before the alerts, November 2026), or whose join failed, have a Discord identity in
`auth` and no link here. Some of them joined the server by the plain invite; the others never did.

- **In the server already: the bot links them, with no action of theirs.** `LinkMembers` asks `auth` every account
  signed in with Discord (`auth.v1.InternalService/GetDiscordUsers`, 100 a page, in account order, an account asked
  deleted left out), and for each one with no link asks Discord whether its user is in the server
  (`GET /guilds/{guild}/members/{user}`, which needs no privileged intent). A member is granted its flag's role and
  linked, as a sign-in's join is (`join_server_usecase`); one that never pledged is linked with no role, and its first
  pledge gives it one.
- **Not in the server: only its own sign-in can add it.** Discord adds a user to a server only with that user's own
  token holding `guilds.join`; the old sign-ins never asked for it, and a token is never kept. The bot cannot write
  to them either, since it shares no server with them. The game already offers them the way in: an account not
  linked sees "Join Discord" in the chat's Discord bar and "Get pinged" when its flag is attacked, and that Discord
  sign-in, a link of the identity it holds already, joins the server (see [Auth](auth.md)). `LinkMembers` counts them
  (`outside`) and changes nothing.
- **Every linked account holds its flag's role** (`moved`): the run asks planet the flag each linked account plays
  for (`GetAllegianceOf`), and one whose link names another moves to it as a pledge does (`follow_flag_usecase`). Those
  are the players of a sold flag who pledged before the alerts knew its name (they hold no role), or a pledge the bus
  gave up on. Discord is not asked whether a linked user is in the server: the grant answers it, and a user who left is
  forgotten.
- **Every linked account is named** (`links.Namer`, below), the ones linked before the nicknames too: the first run
  after they came names every member, and counts each in `named`. Discord is asked only for a member that does not
  wear its name yet, so a second run renames nobody. A member the bot may not rename (the server's owner, a member
  above the bot) is counted in `failed` at each run, its reason logged.
- **A second run links and moves nobody twice**: an account linked already is skipped before Discord is asked whether its user
  is in the server, and a user linked from another account is never moved. The ones outside are asked again: they may have joined since. **An account
  the operator unlinked, still in the server, is linked again**: unlinking stops nothing once a run follows.
- **One account that fails does not stop the others**: it is counted (`failed`) and logged with its reason
  (`log_link_members`), and a run again tries it. The run stops only when `auth` does not answer or the call is cut
  short. It is sequential: a few hundred accounts take a minute or two, within the backoffice's long timeout.
- **It is an operator's run**, from the backoffice's Ops page ("Link Discord members"), not a boot runner: a boot that
  linked every member would undo each unlink an operator made to try the offer.

## The nickname (`links.Namer`)

**A linked member's nickname in the server is its account's username in the game**, so the players on Discord are
the ones they know from the chat and the boards. Discord gives a bot no field of its own on a member: the server
nickname is the one place a name can go, so the bot owns it, and the operator takes **Change Nickname** off
`@everyone` so nobody wears another name (the server's owner and the moderators keep theirs, see below).

- **`links.Namer.Name(account)`** reads the link, asks player the name it shows for the account now
  (`links/rpc_player_names`, `player.v1.InternalService/GetPlayers`), and when the user does not wear it yet sets the
  nickname (`PATCH /guilds/{guild}/members/{user}`, `nick`) and keeps it on the link (`Store.SaveName`, the name
  column alone, so a nickname given while the flag moves keeps the flag's move; and `SaveLinks` keeps the name of a
  link saved with the same user, so the flag's move saved after keeps the nickname). True when the nickname changed.
- **No name takes the nickname off** (`nick` null), and Discord shows the user's own name again: a guest has its code
  alone, and an account its player asked to delete is hidden from the other players, as it is in the game.
- **Who asks it**: the join (above), `LinkMembers` (above), and three subscribers of player's events, through
  `name_member_usecase` and `nickname_subscriber`: `alerts-names` (`player.v1.PlayerNamed`, a name drawn at the first
  sign-in, chosen by the player or given by the operator), `alerts-names-leaving` (`PlayerLeaving`) and
  `alerts-names-returns` (`PlayerReturned`). **The events name the account alone, and the namer asks player the name
  now**, so whichever of them comes last, from whichever stream, leaves the nickname right, and one told twice changes
  nothing.
- **A user who left the server is forgotten** (`10007 Unknown Member`), as when its flag moves.
- **The server's owner, and a member whose highest role is above the bot's, cannot be renamed by any bot**: Discord
  answers `50013 Missing Permissions`, `discord.ErrRefused`. The subscribers log it at Warn and try no more; the link
  keeps no name, so its next rename and each `LinkMembers` try again. Discord refuses a nickname it takes for no name
  the same way; a game username (3 to 15 letters, digits, `_` and spaces) is well within its 32 characters.

## The roles (`roles/`)

**The bot makes a flag's role the first time one is needed**, mentionable, below its own role, in a color of its own,
and `roles` keeps its id and its color by key, `flag:<code>`. A role's name is the flag's English name
(`golang.org/x/text/language/display`), so a new name in the library changes nothing that is kept.
`roles.Book` makes each role once in one process: two subscribers that need the same new one wait on each other.

- **A sold flag's role wears the name its holder gave it** (below), and follows the operator's rename.
- **The color is the flag's hue** (`roles.ColorOf`): FNV-1a of the code, at saturation 62% and lightness 55%, so it is
  bright on Discord's dark theme, the same on every boot, and no two of the common flags share one. Discord draws a
  role with no color in gray, and the flags' own colors are mostly the same red, white and blue. A better palette is a
  change of `ColorOf` alone: the boot repaints.
- **The boot tidies** (`tidy_roles_runner`, `Book.Tidy`), once: a flag role kept in another color is repainted
  (`PATCH /guilds/{guild}/roles/{id}`), a sold flag's kept under another name renamed, and a role the game no longer
  names is deleted from the server and forgotten: that is how the `Battles` role of the first version went, and the
  roles of Brittany, England, Scotland and Wales when they left the build (a sold flag never heard has none). A role
  someone deleted by hand is made again when needed.
- **No role for battles.** A battle is everyone's, whatever flag they play for: its ping is `@everyone`.

### The sold flags' roles

**A flag sold since the game shipped has a role as one the game ships with**, since 2026-11-28 (until then
`nations.FlagOf` knew the built-in flags alone, so a pledge to Heiliges Römisches Reich took the player's role off).
Its name is not the library's: the alerts keep it (`nations.Catalog`).

- **The alerts hear the flags module's names** (`alerts-flags-opened`, `flags.v1.FlagOpened`, at each opening, claim and
  return, and again for each open flag at every boot; `alerts-flags-renamed`, `FlagRenamed`, the operator's rename),
  through `flag_named_subscriber` and `name_flag_usecase`, into `alerts.sold_flags` (`flag`, `name`, `named_at`,
  migration `20261128085523_sold_flags`). **The latest name holds**: a name told at or after the time of the one kept
  replaces it (`nations.Sold.Named`), so an opening heard after the rename it came before changes nothing, and the
  restatement at each boot, which tells the name now at the opening's time, puts right a rename the alerts missed.
- **Held in memory, written through** (`nations.Load` at boot): every take of a batch asks the catalog, and the module
  runs in one process. A flag the game ships with is never read from it. A sold flag no fact named (Bavaria, closed
  before the alerts listened: the restatement tells a closed flag as `FlagClosed` alone) has no role, as before.
- **The role is made at its first grant**, a pledge, a join or `LinkMembers`, under the name kept, which `roles.name`
  keeps beside it (empty for a flag the game ships with, whose name never follows). **A rename renames it**
  (`Book.Refresh`, `PATCH /guilds/{guild}/roles/{id}` with `name`), asked after every fact that names the flag, so a
  rename Discord refused is tried again with the fact, and the boot's tidy renames what is left; a flag with no role yet
  makes none.
- **A closed flag keeps its role.** Its players pick another flag at once, and their role follows the pledge; one that
  comes back finds its role. A removed flag is told closed too, so its role stays: delete it by hand in the server if
  its name must go.
- **The players who pledged before the deploy hold no role** (their link names none): the operator's `LinkMembers`
  moves them to it (above), once after the deploy.

## The pings (`pings/`)

**An attack** is one flag taking 200 of another's tiles within 5 minutes (`pings.Watch`, in memory, per victim). Its
ping goes to `#alerts`, with the victim's role, the attacker, the tiles and the country where most of them were:

> @Ireland 🇬🇧 United Kingdom took 240 of your tiles in 🇺🇸 United States in 5 minutes.
> https://clickplanet.lol/play?lat=40.42&lon=-100.00&zoom=3.0

- **The link opens the game on the attack** (`pings.View`): `lat` and `lon` in degrees, `zoom` the globe camera's (1 at
  rest, the whole globe; the game reads the three, see the frontend's [Discord pings](../../frontend/docs/discord-alerts.md)).
  The view looks at the middle of the attacker's takes on the ground the ping names, each from `TakenTile.point`.
  The 90% of them nearest that middle fill 40% of a desktop's half height, which is most of a phone's half width; the
  10% left are stray clicks, which would zoom the view out to the whole globe. The zoom is kept between 1 and 20, where
  a tile is about 27 px on a 900 px high screen: 20 on `earth-1`, and on another board the zoom where a tile is as
  wide (`TilesTaken.board`; see [Sized from the board](board-scale.md)). A battle or a season has no place: its link is the plain page.
- **Replayed on the last week of season 0**, 200 tiles gave about ten pings a day across thirty flags, the busiest one
  a day; 50 tiles gave 75 a day, and a server that pings all day is muted.
- **A flag is pinged at most once in 3 hours and twice a day** (`pings.History`, read from `attack_pings`).
- **Only a flag a player ever linked is pinged**: no role, nobody to call.
- **A flag is named with its emoji, then its name** (`nations.Label`): the regional indicators of a flag the game ships
  with, the black flag with the tags of `gb-eng`, `gb-sct` or `gb-wls`, and none for any other sold flag (Brittany's
  `xb` would draw two letters):

> @Austria Heiliges Römisches Reich took 230 of your tiles in 🇦🇹 Austria in 5 minutes. https://…
- **An attack older than 10 minutes is not pinged**: a boot or a new group that reads a backlog of takes stays quiet.
- **A restart forgets the watch**: the takes of the last minutes, and at most one attack with them.
- **A ping Discord refused is not tried again.** The ping is kept in the transaction of the post, so a refused post
  keeps nothing; but the watch already counted the attack, and a second delivery of the take finds none.

**A start** is `gamemaster.v1.BattleStarted` (`alerts-battles`) or `SeasonStarted` (`alerts-seasons`), told once
each by the gamemaster (see [The gamemaster](gamemaster.md)). Its ping goes to `#battles`, to everybody in the server,
linked or not:

> @everyone ⚔️ The battle is on: 2x points until 21:00. https://clickplanet.lol/play

The end is `<t:…:t>`, which Discord draws in each reader's own time. The finale says "The Final Battle of season N",
a season "Season N has started on a blank map". **The world's switch of board is told as a start too**: planet's
`BoardSwitched` (`alerts-board-switches`, `board_switched_subscriber`; an arena's map is no switch) is
`pings.WorldGrew`, "🌍 The world grew: 4x more ground. Every country kept its land.", the times the new board's tiles
over the old's, told once by the board (`world/earth-2`); see [Boards and eras](boards-and-eras.md#the-worlds-switch-of-board-switchboard). A start is told once by its round's or book's id (`start_pings`), in
the transaction of the post: a post Discord refused is tried again as the event comes again, until the start is 10
minutes old.

**A post pings the audience it names, whatever the text holds** (`discord.Audience`, sent as `allowed_mentions`): one
role, or `everyone`. The bot needs **Mention @everyone, @here, and All Roles** for the second; without it Discord
posts the text and pings nobody.

## The staff's channel

**The owner is told in a private staff channel of each flag that waits on a person** (the owner, 2026-11-27), by the
game's bot (`reviews/`): one post when a player asks a person to look again at the judge's rejection, and one when a
**paid** flag waits because the judge was unsure, in either order (see [The flags module](flags.md#ask-a-person)).

- **It hears `flags.v1.ReviewAsked`** (`alerts-reviews`, `review_asked_subscriber`), which the flags module records in
  its outbox. A cause this build does not know is left.
- **Each post names the flag, the cause, the judge's why and a link to the submission on the backoffice**
  (`<backofficeUrl>/#/submissions/<id>`), and pings nobody (`discord.Nobody()`, `allowed_mentions` empty): the
  channel's readers see it.

> 🙋 **Smurfland**: its player asks a person to look again at the judge's rejection. Judge: “a cartoon's village”
> https://backoffice.clickplanet.lol/#/submissions/…

> 🤔 **Solandia** is paid and waits on you: the judge was unsure. Judge: “could be fictional” https://…

- **The name and the why are text, not markdown**: Discord's markdown characters are escaped, a line break is a
  space, a why past 300 characters is cut.
- **Told once by its submission and cause** (`review_posts`, `20261127113648_review_posts`), in the transaction of the
  post: a post Discord refused keeps nothing, and the event comes again.
- **Off while `alerts.discord.staffChannelId` is empty**: nothing subscribes, and the events wait in their stream; a
  channel named later is told what the stream still holds. The bot cannot make the channel (it has no **Manage
  Channels**): the owner makes it private, and gives the bot **View Channel** and **Send Messages** in it (see
  `deploy/vps/README.md`, "Discord pings"). Production's is `1575835114892431450` (2026-11-27).
- **A post Discord refuses is logged** ("the alerts module refused an event", "failed to tell the staff of …"):
  `50001 Missing Access` is View Channel missing, `50013 Missing Permissions` is Send Messages missing. The bus tries it
  again for about four minutes, then buries that warning in `events:dead`.

## The Discord client (`discord/rest_discord`)

The bot (`Authorization: Bot …`) on API v10: members (`Join`, `IsMember`, `Grant`, `Revoke`, `Rename`), roles and
posts. The
OAuth trade is `auth`'s. A `4xx` is `discord.ErrRefused` (`ErrNotMember` for `10007`, `ErrNoRole` for `10011`), a `5xx`
a plain error that a subscriber tries again. **A `429` whose `Retry-After` is 5 s or less is waited out and sent
again, twice at most**, so a run of `LinkMembers` rides Discord's per-route limit; a longer wait, or none named, is a
plain error, tried again by its caller as before. The bot needs **Manage Roles**, **Manage Nicknames**, **Create
Instant Invite** (to add a member), **View Channels**, **Send Messages** and **Mention @everyone, @here, and All
Roles**; the setup is in `deploy/vps/README.md`, "Discord pings".

## Config

`alerts.enabled`, `alerts.database.*` (schema `alerts`), `alerts.redis.*`, `alerts.gameUrl` (the page a ping links
to, `https://clickplanet.lol/play` by default), `alerts.backofficeUrl` (where a warning to the staff links,
`https://backoffice.clickplanet.lol` by default), and `alerts.discord.botToken`, `guildId`, `alertsChannelId`,
`battlesChannelId`. On, any of them empty or not a Discord id refuses the boot. `staffChannelId` may be empty (no
warnings to the staff), and is a Discord id otherwise. The bot is `auth.discord`'s
application's: Discord adds a user to a server only with a token of the bot's own application. `auth.discord.joinServer`
is the other half, and is on in production with this module. **Keep it off while this module is off**: nobody hears
the grants, which wait in their stream up to `redis.maxLen`, and a group made later reads them all, each one expired.

## Tests

`migrations/stores_test.go` runs the stores on postgres. The use cases run over the in-memory stores and
`roles.FakeMaker`, `pings.RecordedPoster`; `cpatomic.OnCommit` keeps an in-memory write until the fake transaction
commits. `rest_discord` runs against an `httptest` Discord. `discord_join_granted_subscriber`'s test pins what is
tried again and what is not, and that the token is not logged; `rpc_auth_grants`' what auth's answers mean.
`reviews/review_test.go` pins a warning's text and its escaping, `post_review_usecase`'s test that it is told once per
cause and tried again after a refusal, `review_asked_subscriber`'s the event read. `nations/flag_test.go` pins which flags have a role, their names and emoji, and the latest name kept;
`roles_test.go` a sold flag's role made under its name, renamed once, and kept by the tidy while its flag is heard;
`name_flag_usecase`'s and `flag_named_subscriber`'s tests the facts heard; `link_members_usecase`'s a linked member moved
to its flag's role. `links/namer_test.go` pins the nickname: given once, followed, taken off with no name, a user who left forgotten, a
refusal that keeps the link unnamed, and a name kept without the flag's move undone; `links.FakeNames` and
`links.FakeRenamer` stand in for player and Discord. `nickname_subscriber`'s test pins the three events and what is
tried again; `rpc_player_names`' what player's answer means.
