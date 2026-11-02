# Discord pings (`internal/alerts/`)

**A player links its Discord account once, and the game's Discord server pings it.** The link adds the player to the
server with two roles: `Battles`, and the role of the flag it plays for. The server pings a flag's role when that flag
is attacked, and `Battles` when a battle or a season starts. Off by default (`alerts.enabled`); off, the module is not
built and `alerts.v1` 404s, which the client reads as no offer.

Why: a newcomer who plays once never comes back on its own, and the players who come back on day 2 to 7 have no way to
hear that their land is attacked. They have Discord open on their phone; a website's notifications they do not allow.

## The link (`links/`)

**OAuth 2.0 authorization code with PKCE, over two RPCs, on the sign-in's Discord application and redirect URI.**
Scopes `identify guilds.join`: who the user is, and leave to add them to the server. No new URI to register: the page
tells a link from a sign-in by what it started (see the frontend's
[Discord alerts](../../frontend/docs/discord-alerts.md)).

- **`StartDiscordLink`** draws a `links.Flow` (state and verifier, 32 random bytes each, 10 minutes) for the caller's
  account and keeps it in `link_flows`, one per account: a second start replaces the first. It answers Discord's
  authorization URL.
- **`CompleteDiscordLink(code, state)`** finds the caller's flow, spends it (the state compared in constant time,
  before it expires, once), trades the code with the verifier, reads the user (`/users/@me`), asks planet the flag the
  account pledged (`planet.v1.InternalService/GetAllegianceOf`), makes the roles it needs, adds the user to the server
  with them (`PUT /guilds/{guild}/members/{user}`, or each role granted when the user is in it already), and keeps the
  link. It answers the server's page in Discord.
- **The flow is bound to the account, not to a cookie.** Every call carries the caller's click token, so a code carried
  to another browser finds that account's flow, or none: `FailedPrecondition`. The flow is spent before the code is
  traded, so a code is good once.
- **One Discord user, one account.** A user who links from a second account moves there: the first link is forgotten.
- **The access token is never kept**: it joins the user once. A link keeps the account, the Discord user id, the flag
  of its role, and when.
- **The role follows the flag.** `alerts-flags` hears every `planet.v1.TileTaken` with an account; a linked account
  that took a tile for another flag than its role's is granted the new flag's role and loses the old one. An account
  that never pledged gets its first flag's role on its first take. A user who left the server (`10007 Unknown Member`)
  is forgotten: it asked for no more pings.
- **A deleted account is forgotten** (`alerts-accounts`, `auth.v1.AccountDeleted`): its link and its flow. The Discord
  user stays in the server with its roles: that is its own Discord account.

| answer | when |
|---|---|
| `Unauthenticated` | no click token, or one with no account |
| `FailedPrecondition` | no flow of this account, another state, a spent or a lapsed one: start again |
| `PermissionDenied` | Discord refused the code or answered no user: start again |
| `ResourceExhausted` | `alerts.rateLimiter`, on start and complete: 6 at once, one every 30 s |

`GetDiscordLink` answers `offered` (true: a server that answers at all offers it), `linked`, and the server's page.
It reads `links` straight (a query).

## The roles (`roles/`)

**The bot makes each role the first time one is needed**, mentionable, below its own role, and `roles` keeps its id by
key: `battles`, or `flag:<code>`. A role's name is the flag's English name (`golang.org/x/text/language/display`;
`xb` is Brittany), so a new name in the library changes nothing that is kept. `roles.Book` makes each role once in one
process: two subscribers that need the same new one wait on each other.

## The pings (`pings/`)

**An attack** is one flag taking 200 of another's tiles within 5 minutes (`pings.Watch`, in memory, per victim). Its
ping goes to `#alerts`, with the victim's role, the attacker, the tiles and the country where most of them were:

> @Ireland 🇬🇧 United Kingdom took 240 of your tiles in 🇺🇸 United States in 5 minutes. https://clickplanet.lol/play

- **Replayed on the last week of season 0**, 200 tiles gave about ten pings a day across thirty flags, the busiest one
  a day; 50 tiles gave 75 a day, and a server that pings all day is muted.
- **A flag is pinged at most once in 3 hours and twice a day** (`pings.History`, read from `attack_pings`).
- **Only a flag a player ever linked is pinged**: no role, nobody to call.
- **An attack older than 10 minutes is not pinged**: a boot or a new group that reads a backlog of takes stays quiet.
- **A restart forgets the watch**: the takes of the last minutes, and at most one attack with them.
- **A ping Discord refused is not tried again.** The ping is kept in the transaction of the post, so a refused post
  keeps nothing; but the watch already counted the attack, and a second delivery of the take finds none.

**A start** is `gamemaster.v1.BattleStarted` (`alerts-battles`) or `SeasonStarted` (`alerts-seasons`), told once
each by the gamemaster (see [The gamemaster](gamemaster.md)). Its ping goes to `#battles`, with `Battles`:

> @Battles ⚔️ The battle is on: 2x points until 21:00. https://clickplanet.lol/play

The end is `<t:…:t>`, which Discord draws in each reader's own time. The finale says "The Final Battle of season N",
a season "Season N has started on a blank map". A start is told once by its round's or book's id (`start_pings`), in
the transaction of the post: a post Discord refused is tried again as the event comes again, until the start is 10
minutes old.

**A post pings the one role it names**, whatever the text holds (`allowed_mentions`).

## The Discord client (`discord/rest_discord`)

One client for both halves of the application: the OAuth endpoints, and the bot (`Authorization: Bot …`) on API v10.
A `4xx` is `discord.ErrRefused` (`ErrNotMember` for `10007`), a `429` or a `5xx` a plain error that a subscriber
tries again. The bot needs **Manage Roles**, **Create Instant Invite** (to add a member), **View Channels** and
**Send Messages**; the setup is in `deploy/vps/README.md`, "Discord pings".

## Config

`alerts.enabled`, `alerts.database.*` (schema `alerts`), `alerts.redis.*`, `alerts.gameUrl` (the page a ping links
to, `https://clickplanet.lol/play` by default), `alerts.rateLimiter.*`, and `alerts.discord.clientId`,
`clientSecret`, `botToken`, `redirectUrl` (the sign-in's), `guildId`, `alertsChannelId`, `battlesChannelId`. On, any
of them empty or not a Discord id refuses the boot.

## Tests

`migrations/stores_test.go` runs the three stores on postgres. The use cases run over the in-memory stores and
`roles.FakeMaker`, `pings.RecordedPoster`; `cpatomic.OnCommit` keeps an in-memory write until the fake transaction
commits. `rest_discord` runs against an `httptest` Discord.
