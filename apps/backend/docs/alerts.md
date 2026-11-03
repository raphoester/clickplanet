# Discord pings (`internal/alerts/`)

**A player connects Discord once, by signing in with it, and the game's Discord server pings it.** The sign-in adds the
player to the server with the role of the flag it plays for. The server pings a flag's role when that flag is
attacked or places in a day's top three (the day's ranking), and everybody in the server (`@everyone`) when a battle or
a season starts. Off by default (`alerts.enabled`); off, the module is not built, `alerts.v1` 404s, which the client
reads as no offer, and a Discord sign-in joins no server.

Why: a newcomer who plays once never comes back on its own, and the players who come back on day 2 to 7 have no way to
hear that their land is attacked. They have Discord open on their phone; a website's notifications they do not allow.

**One Discord connection, not two** (the owner, 2026-11-02). The first version had its own Discord trip, "Get pinged",
beside "Sign in with Discord": two consents on the same application, and nobody could say why both were needed. Now
the alerts module starts no trip of its own: `auth`'s Discord sign-in asks the leave to join the server too, and hands
alerts the token (see [Auth](auth.md)).

## The link (`links/`)

**Made by `InternalService/JoinServer(account_id, discord_user_id, access_token)`**, which `auth` calls once a Discord
sign-in or link is saved, when `auth.discord.joinServer` is on (Discord's scope `guilds.join`). It is served on the
internal listener alone.

- **`join_server_usecase`** asks planet the flag the account pledged (`planet.v1.InternalService/GetAllegianceOf`),
  makes its role if needed, adds the user to the server with it (`PUT /guilds/{guild}/members/{user}`, or the role
  granted when the user is in it already), and keeps the link. An account that never pledged joins with no role: its
  first pledge gives it.
- **An account linked to this user already changes nothing**: every sign-in calls `JoinServer`, and a user who left the
  server since is not added back by each one.
- **One Discord user, one account.** A user linked from another account moves here: the first link is forgotten.
- **The access token is never kept**: it joins the user once. A link keeps the account, the Discord user id, the flag
  of its role, and when.
- **A failure never fails the sign-in.** `auth` logs it (`rpc_alerts_server`), and the player is signed in, not linked:
  the game offers to join again. `PermissionDenied` is Discord refusing the token (no `guilds.join`).
- **The role follows the flag pledged.** `alerts-allegiances` hears `planet.v1.AllegiancePledged`, told when a pledge
  sets or changes the flag (see [Allegiance](allegiance.md)): a linked account is granted the new flag's role and loses
  the old one. Before, the role followed the flag of the next tile taken, so a player who changed flag and did not
  click kept being pinged for the old one. A user who left the server (`10007 Unknown Member`) is forgotten: it asked
  for no more pings.
- **A deleted account is forgotten** (`alerts-accounts`, `auth.v1.AccountDeleted`). The Discord user stays in the
  server with its roles: that is its own Discord account.

`GetDiscordLink` answers `offered` (true: a server that answers at all offers it), `linked`, and the server's page
(`#alerts`). It reads `links` straight (a query), with the caller's click token. The public route,
`/alerts.v1.AlertsService/*`, is listed in `deploy/vps/caddy/Caddyfile`: without it Caddy answers 404, and the game
shows no offer.

### The operator (`AdminService`)

On the console and admin listeners, as every operator tool (see [Operator tools](admin.md)):

- **`GetPlayerLink(account_id)`**: the account's link (the Discord user, its role's flag, when), or none.
- **`UnlinkPlayer(account_id)`**: forgets the link and takes the flag's role off the user (`unlink_player_usecase`).
  The user stays in the server; a user who left it already is unlinked all the same. `unlinked` false: there was no
  link. A role Discord would not take off keeps the link (`FailedPrecondition`). Unlinked, the game offers that account
  to join again, and its next Discord sign-in joins it again: this is how the offer is tried. Players cannot unlink
  themselves; leaving the server is how they stop the pings.

## The roles (`roles/`)

**The bot makes a flag's role the first time one is needed**, mentionable, below its own role, in a color of its own,
and `roles` keeps its id and its color by key, `flag:<code>`. A role's name is the flag's English name
(`golang.org/x/text/language/display`; `xb` is Brittany), so a new name in the library changes nothing that is kept.
`roles.Book` makes each role once in one process: two subscribers that need the same new one wait on each other.

- **The color is the flag's hue** (`roles.ColorOf`): FNV-1a of the code, at saturation 62% and lightness 55%, so it is
  bright on Discord's dark theme, the same on every boot, and no two of the common flags share one. Discord draws a
  role with no color in gray, and the flags' own colors are mostly the same red, white and blue. A better palette is a
  change of `ColorOf` alone: the boot repaints.
- **The boot tidies** (`tidy_roles_runner`, `Book.Tidy`), once: a flag role kept in another color is repainted
  (`PATCH /guilds/{guild}/roles/{id}`), and a role the game no longer names is deleted from the server and forgotten.
  That is how the `Battles` role of the first version went. A role someone deleted by hand is made again when needed.
- **No role for battles.** A battle is everyone's, whatever flag they play for: its ping is `@everyone`.

## The pings (`pings/`)

**An attack** is one flag taking 200 of another's tiles within 5 minutes (`pings.Watch`, in memory, per victim). Its
ping goes to `#alerts`, with the victim's role, the attacker, the tiles and the country where most of them were:

> @Ireland 🇬🇧 United Kingdom took 240 of your tiles in 🇺🇸 United States in 5 minutes.
> https://clickplanet.lol/play?lat=40.42&lon=-100.00&zoom=3.0

- **The link opens the game on the attack** (`pings.View`): `lat` and `lon` in degrees, `zoom` the globe camera's (1 at
  rest, the whole globe; the game reads the three, see the frontend's [Discord pings](../../frontend/docs/discord-alerts.md)).
  The view looks at the middle of the attacker's takes on the ground the ping names, each from `TileTaken.point`.
  The 90% of them nearest that middle fill 40% of a desktop's half height, which is most of a phone's half width; the
  10% left are stray clicks, which would zoom the view out to the whole globe. The zoom is kept between 1 and 20, where
  a tile is about 27 px on a 900 px high screen. A battle or a season has no place: its link is the plain page.
- **Replayed on the last week of season 0**, 200 tiles gave about ten pings a day across thirty flags, the busiest one
  a day; 50 tiles gave 75 a day, and a server that pings all day is muted.
- **A flag is pinged at most once in 3 hours and twice a day** (`pings.History`, read from `attack_pings`).
- **Only a flag a player ever linked is pinged**: no role, nobody to call.
- **An attack older than 10 minutes is not pinged**: a boot or a new group that reads a backlog of takes stays quiet.
- **A restart forgets the watch**: the takes of the last minutes, and at most one attack with them.
- **A ping Discord refused is not tried again.** The ping is kept in the transaction of the post, so a refused post
  keeps nothing; but the watch already counted the attack, and a second delivery of the take finds none.

**A start** is `gamemaster.v1.BattleStarted` (`alerts-battles`) or `SeasonStarted` (`alerts-seasons`), told once
each by the gamemaster (see [The gamemaster](gamemaster.md)). Its ping goes to `#battles`, to everybody in the server,
linked or not:

> @everyone ⚔️ The battle is on: 2x points until 21:00. https://clickplanet.lol/play

The end is `<t:…:t>`, which Discord draws in each reader's own time. The finale says "The Final Battle of season N",
a season "Season N has started on a blank map". A start is told once by its round's or book's id (`start_pings`), in
the transaction of the post: a post Discord refused is tried again as the event comes again, until the start is 10
minutes old.

**A day's end** is `gamemaster.v1.DayEnded` (`alerts-days`), told once the day is scored (see [Hearing
scoring](gamemaster.md#hearing-scoring)). Its post, the day's ranking, goes to `#alerts`, and pings the role of each
flag placed 3rd or better that day, ties included (`pings.Podium`):

> 🏁 **Day 3 of season 1 is over.** The points it gave:
> 🥇 🇮🇪 @Ireland +25
> 🥈 🇫🇷 @France +18
> 🥉 🇬🇧 United Kingdom +15
> 4️⃣ 🇪🇸 Spain +12
> …
> 🏆 **The season's ranking:**
> 🥇 🇫🇷 France 118 points
> 🥈 🇮🇪 Ireland 97 points
> …
> https://clickplanet.lol/play

- **The first list is the day's top 10 and the points each won**; the second, the season's table once they were granted
  (grants included), down to the 10th place, ties included (`pings.TableRows`). `announce_ranking_usecase` reads both by
  the round's id from `scoring.v1.InternalService/GetRound` on the internal listener (`rpc_scoring_rounds`, 5 seconds):
  its results and its `after`. A round scoring has not scored (`ended_at` unset, live standings) is an error, tried again.
- **A flag of the top three no player ever linked is named, not pinged**: it has no role. A day whose top three
  nobody linked still posts, and pings nobody.
- **A place is a medal for 1st to 3rd, then a keycap (4️⃣ to 🔟)**: "4." at a line's start is a Markdown list, which
  Discord numbers again, so two flags tied 2nd would read 2 and 3.
- **Told once by the round's id** (`ranking_pings`), in the transaction of the post: a post Discord refused, or a round
  scoring could not answer for, is tried again as the event comes again, **until the day ended an hour ago**
  (`pings.RankingFresh`). A new consumer group reads the stream from its start, so without it the first boot would post
  every day of the season. A day nobody scored in (no tile held) posts nothing. Only a day: a battle's end posts
  nothing.

**A post pings the audience it names, whatever the text holds** (`discord.Audience`, sent as `allowed_mentions`): some
roles (none pings nobody), or `everyone`. The bot needs **Mention @everyone, @here, and All Roles** for the second; without it Discord
posts the text and pings nobody.

## The Discord client (`discord/rest_discord`)

The bot (`Authorization: Bot …`) on API v10: members, roles and posts. The OAuth trade is `auth`'s. A `4xx` is
`discord.ErrRefused` (`ErrNotMember` for `10007`, `ErrNoRole` for `10011`), a `429` or a `5xx` a plain error that a
subscriber tries again. The bot needs **Manage Roles**, **Create Instant Invite** (to add a member), **View
Channels**, **Send Messages** and **Mention @everyone, @here, and All Roles**; the setup is in `deploy/vps/README.md`,
"Discord pings".

## Config

`alerts.enabled`, `alerts.database.*` (schema `alerts`), `alerts.redis.*`, `alerts.gameUrl` (the page a ping links
to, `https://clickplanet.lol/play` by default), and `alerts.discord.botToken`, `guildId`, `alertsChannelId`,
`battlesChannelId`. On, any of them empty or not a Discord id refuses the boot. The bot is `auth.discord`'s
application's: Discord adds a user to a server only with a token of the bot's own application. `auth.discord.joinServer`
is the other half, and is on in production with this module.

## Tests

`migrations/stores_test.go` runs the stores on postgres. `pings_test.go` pins the day's ranking word for word (ties,
the top three's roles, a flag nobody linked, the table cut at the 10th place), and `rpc_scoring_rounds`' test the read
of a scored round and the refusal of one that runs. The use cases run over the in-memory stores and
`roles.FakeMaker`, `pings.RecordedPoster`; `cpatomic.OnCommit` keeps an in-memory write until the fake transaction
commits. `rest_discord` runs against an `httptest` Discord.
