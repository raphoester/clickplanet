# Chat moderation

**Claude moderates the chat on its own, every 2 hours, by the rules below**, through the ClickPlanet connector (see
[Claude as a connector](../apps/backend/docs/connector.md)). This file is the policy: every run reads it first, so a
change here changes what the runs do. It holds rules only. **What a run did, and to whom, is never written here**: the
repository is public.

**The game wants players, so moderation is light.** A mute leaves the player the map, but it silences its whole network
too (the address over IPv4, the /64 over IPv6: a school, a mobile carrier), and the chat announces it. Only the operator
mutes an account alone (`account_only`), for a network that people who did nothing share. So a removal
comes first, a mute only stops a spree, and nothing goes past 24 hours without the operator.

## What stays

- **Political talk**, unless it attacks a person or a group of people, or praises atrocities.
- **Swearing and insults**, in any language, short of the severe content below.
- **Insults aimed at a country or a people**: the rivalry is the game.
- **Political usernames.**

## What goes

- **Severe content**: a slur, a Nazi symbol or slogan, a threat, sexual content or a sexual remark aimed at a player,
  someone's personal information.
- **Hateful politics**: an attack on a person or a group of people, praise of atrocities.
- **Spam**: the same message again and again, and links or ads for other games or servers. Removed each time, and an
  offence only when it goes on after a removal.
- **A shock GIF**: gore, porn, Nazi or shock imagery.

## The steps

Counted per account from its **moderation history**, the chat and the GIFs apart (each is a mute of its own,
`MUTE_KIND_CHAT` and `MUTE_KIND_GIFS`; a removed GIF counts in the GIFs):

| Offence           | What the run does                 |
|-------------------|-----------------------------------|
| 1st               | removes the message               |
| 2nd               | removes it, mutes 15 minutes      |
| 3rd               | removes it, mutes 1 hour          |
| 4th and after     | removes it, mutes 24 hours        |
| after a 24h mute  | removes it, and the operator decides |

- **The slate is wiped after 7 days with no offence.** A run reads the account's last 7 days
  (`chat_list_moderation` with its `accountId` and `since`), and the count is the offences there since the last gap of
  7 days.
- **Every action carries its reason, and the reason says what it counts**: `offence N (chat|gifs): <what>` for an
  offence, `spam, no offence` for a first spam removal, `operator: <what>` for what the operator asked. The history
  keeps the reason and who acted (Claude, or the operator), so the next run counts from it and the operator audits it.
- **A severe offence goes straight to the 3rd step**, whatever the count: removed, muted 1 hour, and flagged to the
  operator. The next offence is the 24-hour one.
- **No run mutes for longer than 24 hours.** Longer is the operator's call.
- **A GIF offence takes the GIF mute**, so the player keeps the text chat. A porn or gore GIF is severe.
- A removed message stays in the chat as evidence until the chat's retention (30 days, `chat.storage.retention`) drops
  it. Its text, or its GIF's address, stays in the moderation history for 90 days, without the sender's address.

## Usernames

- **Renamed**: a slur, a Nazi reference (`Hitler88`), a sexual name. The run gives a neutral name, `Player` and
  digits, with the reason, and lists the rename in the digest (`player_list_renames` keeps it 90 days, with the old
  name). The player can choose another name at once.
- **Kept**: every other name, political ones included.

## Reports

- **A report is read with its conversation**: the messages before and after it, in its room, and the reported
  player's own messages. Never on the reported line alone.
- **Each is decided with a one-line reason**: acted when it led to an action of the steps, dismissed otherwise. One
  whose message is past the chat's retention (30 days) is dismissed with that reason.
- **A reporter is never punished for reporting.** One who files many baseless reports is named in the digest.
- **A cheating report is inspected** with the antibot (`antibot_inspect_player`), a ban or a revert is proposed to
  the operator, and the report is left open for the operator to decide.

## Passed to the operator, never acted on alone

- **Child safety** (an adult asking a kid for its age, photos or social accounts) **and self-harm**: pushed to the
  operator's phone at once, and nothing else is done.
- **Cheating**: above.
- **A player who goes on after a 24-hour mute.**
- **Every ban, unban, revert, and mute over 24 hours.**
- **A case the run is unsure of**: left alone, and listed in the digest.

## How a run judges

- **Every language, the same rules.**
- **The conversation decides, never a word**: a line that reads as an insult alone can be banter among rivals.
- **The operator's own account is never acted on.**

## How a run works

- **Every 2 hours**, a scheduled Claude session with the ClickPlanet connector, in the operator's `clickplanet` cloud
  environment, whose network allows the GIF host (`static.klipy.com`), so a run sees each GIF: the open reports first
  (`support_list_reports` with `openOnly`), then every message since the last run (`chat_list_messages`, paged back
  until `sentAt` passes the last run), then, for each account it acts on, its last 7 days of history
  (`chat_list_moderation`), then the actions these rules call for, each with its reason.
- **The tools a run calls without asking**: `chat_remove_message`, `chat_mute` (24 hours at most), `chat_unmute`,
  `support_decide_report`, `player_rename_player`. Every other write is a proposal in the digest. The operator allows
  those five on claude.ai and leaves the rest on "Needs approval". The reads (`chat_list_moderation`,
  `player_list_renames` and the others) are read-only tools, called without asking.
- **A push to the operator's phone** for child safety, self-harm, and anything that waits on the operator. **A daily
  digest** of the rest: the actions and their reasons (every action since the last digest: `chat_list_moderation` and
  `player_list_renames` with `since`, Claude's told from the operator's by their `client`), the reports decided, the renames, the repeat reporters, the
  proposals still open, and whether the players muted kept playing (their takes in the ledger after the mute), so the
  steps can be made lighter if mutes cost players.

## Not built yet

- **The scheduled task**, and the five tools allowed on claude.ai.
