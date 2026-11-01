# The player page

`src/players/`. **A player is an account or an address** (`Caller` in `caller.ts`), never both: the planet's tools
take either. An account id is a UUID; anything else typed is an address, which the server reads as its scope (the
address over IPv4, the /64 over IPv6) and checks. The player open is in the address bar (`#/players/<account id or
address>`), so Back goes to the list, as it was.

- **One search box** (`PlayerList`, `searchOf`): an account id opens the account, an address (IPv4, or hex with a
  colon, with or without a `/64`) opens the address, and anything else is a name: `player.v1.AdminService/SearchPlayers`,
  every account whose username or guest code holds it, any case, the same name first. It reaches past the ledger's
  72 hours, so a banned player the lists no longer show is found by the name the chat gives it. A name row is its
  name and whether it is a username or a guest code; it carries no takes, since the ledger is not asked.
- **The list** is a table: `TopPlayers` (most takes first, then most tiles held), or, with a **Flag** and an optional
  ground and **Filter**, `FindPlayers`. "× Back to the most active" clears both. 20 by default, up to 500. Each row of
  the ledger's 72 hours is one account on every address it took from, shown with the address of its latest take (or
  one address, for takes with no account): its takes, the tiles it still holds and their share of its takes (many
  takes and few tiles is a player painted over as fast as it paints), its takes a minute, how long it played, its
  latest take and any ban (a red edge too). **A column head sorts the rows read**, most first, then least first;
  unsorted is the server's order. The whole row opens the player. Below 720px the address and the rates are hidden,
  so nothing scrolls sideways. The list is read again on the way back from a player, as the reports are.
- **A row is shown by its flag and its name, never its account id** (`PlayerName`): the flag it took the most tiles
  for (`flag_country_id`), as an emoji and a name, then its username or its guest code. The planet's rows carry no
  name, and planet may not ask `player` (the backend's [Player](../../backend/docs/player.md)), so the page asks
  `player.v1.AdminService/GetAuthors` for the accounts of the page once the list answers (`withAuthors`). An account
  the game never named is "Unnamed": it never joined the roster or posted, and the game's page joins the roster as
  soon as it may click, so it is most likely a script. A row with no account is "No account". **Names that cannot be
  read leave the list listed**, every account "Unnamed", with the reason above it.
- **The player page** (`PlayerView`), top down: its flag and its name (or its address), **where it stands**
  (`Standing`: banned until when and the offence, the jury's opinion, a pardon, and for an account muted until when,
  from `chat.v1.AdminService/GetMute`, "by a mute on its network" when another account's mute holds it), its account
  and its address with Copy,
  the other half opening from there, and its row's numbers; then **the tools**; then its profile and the antibot's
  reading side by side from 1100px; then what it said, folded.
- **The other half of a player**: the address of an account comes from its row, or else from where the antibot last
  read it; the account of an address from its row. **A guest the jury banned is banned on both**, and an unban lifts
  one key.
- **An account's page reads its profile** (`player.v1.AdminService/GetPlayer`, `ProfilePanel`): signed in or a
  guest, when it was made, its name and whether it is a username or a guest code, the crown, its color, the tiles
  it took, its streak, its messages, its titles (the one worn marked), and every country it plays for and against,
  five of each and the rest behind "All". A guest has all of it too, which the game's public card does not. The
  heading is its flag and its name: the name the profile answers, which is the name it holds now, over the one the
  page that opened it knew; the flag of its row, or else the country it took the most tiles for. `not_found` is an
  account the game keeps nothing about. An address has no profile, and is not asked.
- **An account's page shows what it said, in its conversations** (`chat/TheirMessages`, as a report does), folded
  under "What Ana said" with its count of messages: it reads only once opened. Its lines are marked (`line-theirs`),
  and no line is tagged: there is no report. An address has no messages, since the chat keeps them by account, and is
  not asked.
- **The page inspects on open** (`antibot.v1.AdminService/InspectPlayer`, `readingOf`): the antibot's readings, its
  ban, its pardon and its clicks. The antibot off (its service absent, `unimplemented`) reads as such, not as a
  failure. A ban or an unban reads it again, and a mute or an unmute reads `GetMute` again.
- **The ban tools are the antibot's** (`antibot.v1.AdminService`, the `antibot` client), since the antibot became a
  module of its own; `planet.v1` keeps the three calls for a deploy, answered by the antibot, then drops them. A ban
  reaches planet a moment after its answer.
- **The tools, one at a time behind tabs**, so the page stays short: Ban (`BanPlayer`, the ladder's next step by
  default), Unban (`UnbanPlayer`, which forgets the latest offence and pardons the key, and says until when;
  `not_found` is no ban left), Revert (`RevertPlayer`, dry run first), and for an account alone Mute (`Mute`), Unmute
  (`Unmute`, which ends every mute given to the account and says until when it was muted; `not_found` is no mute of
  its own: a mute on its network given to another account is lifted from that account's page),
  Charges (`GrantCharges`: a refill, a bomb, enclosures, spread clicks, shields; the answer says what the account
  held before and after, since the caps hold) and Flag (`FlagPanel`, `ReleaseAllegiance`: the hour of wait after a
  change of flag is lifted, so the player's next pick goes at once; the answer says when a battle's lock still holds
  it). **Max out** sends every charge at the form's ceiling (`MAX_GRANT`) in one tap, and the server's caps make it a
  full inventory.
