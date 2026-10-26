# The player page

`src/players/`. **A player is an account or an address** (`Caller` in `caller.ts`), never both: the planet's tools
take either. An account id is a UUID; anything else typed is an address, which the server reads as its scope (the
address over IPv4, the /64 over IPv6) and checks.

- **The list** (`PlayerList`): `TopPlayers` (most takes first, then most tiles held), or **By flag**,
  `FindPlayers` with a flag and an optional ground, or **By name**. 20 by default, up to 500. Each row of the ledger's 72 hours is
  one account on every address it took from, shown with the address of its latest take (or one address, for takes
  with no account), with its takes, the tiles it still holds and their share of its takes (many takes
  and few tiles is a player painted over as fast as it paints), its rates and any ban. The list is read again on
  the way back from a player, as the reports are.
- **By name** (`NameResults`) is `player.v1.AdminService/SearchPlayers` with what the operator typed: every account
  whose username or guest code holds it, any case, the same name first. It reaches past the ledger's 72 hours, so a
  banned player the lists no longer show is found by the name the chat gives it. A row is its name and whether it is
  a username or a guest code, and opens its account (`MatchCard`); it carries no takes, since the ledger is not
  asked.
- **A row is shown by its flag and its name, never its account id** (`PlayerName`): the flag it took the most tiles
  for (`flag_country_id`), as an emoji and a name, then its username or its guest code, then its address. The
  planet's rows carry no name, and planet may not ask `player` (the backend's [Player](../../backend/docs/player.md)), so
  the page asks `player.v1.AdminService/GetAuthors` for the accounts of the page once the list answers
  (`withAuthors`). An account the game never named is "Unnamed": it never joined the roster or posted, and the
  game's page joins the roster as soon as it may click, so it is most likely a script. A row with no account is "No account". **Names
  that cannot be read leave the list listed**, every account "Unnamed", with the reason above it.
- **Open a player** from a row (its account, or its address when it has none), from the field at the top, from a
  report, or from a note. The other half of a row (the address of an account, the account of an address) opens from the
  header, and the inspection's address of an account's latest take opens too: **a guest the jury banned is banned
  on both**, and an unban lifts one key.
- **An account's page reads its profile** (`player.v1.AdminService/GetPlayer`, `ProfilePanel`): signed in or a
  guest, when it was made, its name and whether it is a username or a guest code, the crown, its color, the tiles
  it took, its streak, its messages, its titles (the one worn marked), and every country it plays for and against,
  five of each and the rest behind "All". A guest has all of it too, which the game's public card does not. The
  heading is its flag and its name: the name the profile answers, which is the name it holds now, over the one the
  page that opened it knew; the flag of its row, or else the country it took the most tiles for. `not_found` is an
  account the game keeps nothing about. An address has no profile, and is not asked.
- **An account's page shows what it said, in its conversations** (`chat/TheirMessages`, as a report does): its
  messages by its account id, the chat around each, three conversations at a time. Its lines are marked
  (`line-theirs`), and no line is tagged: there is no report. An address has no messages, since the chat keeps them
  by account, and is not asked.
- **The page inspects on open** (`antibot.v1.AdminService/InspectPlayer`): the antibot's readings, its ban, its pardon
  and its clicks. The antibot off (its service absent, `unimplemented`) reads as such, not as a failure. A ban or an
  unban reads it again.
- **The ban tools are the antibot's** (`antibot.v1.AdminService`, the `antibot` client), since the antibot became a
  module of its own; `planet.v1` keeps the three calls for a deploy, answered by the antibot, then drops them. A ban
  reaches planet a moment after its answer.
- **The tools**: ban (`BanPlayer`, the ladder's next step by default), unban (`UnbanPlayer`, which forgets the
  latest offence and pardons the key, and says until when; `not_found` is no ban left), revert (`RevertPlayer`, dry run first), and for an account alone
  mute (`Mute`) and charges (`GrantCharges`: a refill, a bomb, enclosures, spread clicks, shields; the answer
  says what the account held before and after, since the caps hold).
