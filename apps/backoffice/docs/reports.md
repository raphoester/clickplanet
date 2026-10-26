# Acting on a report

`src/reports/` is the Reports page: `ReportsView` shows the list, or one report's detail.

- **The list shows the open reports first** (`ListReports` with `open_only`), and a switch shows them all. A
  decided report shows its outcome, comment and time (`DecisionLine`). The list is read again on the way back
  from a detail, so a report just closed leaves the open list.
- **The detail** (`ReportDetail`): who reported whom, each by name and account id, the reason and the text. With
  a `message_id` it reads the message in its conversation (`GetMessage`, `MessagePanel`), struck through with
  "Removed at …" once removed. A message past the chat's retention is `not_found`: "This message is no longer
  kept."
- **The conversation** (`Conversation`): a message alone cannot be judged, so the page pages through the chat
  around it with `ListMessages`, the message's id as the cursor: the 10 messages before it and the 10 after
  (`FIRST_PAGE`), at once, then 20 more each time "Show earlier" or "Show later" is pressed (`NEXT_PAGE`), from
  the oldest or the newest message shown. A page shorter than asked is the end, and its button goes. The reported
  message shows while they load, marked ("Reported message", `aria-current`), and stays shown if they fail.
  `GetMessage` names no author, so its line takes the name the report gives its account.
- **What the reported account said, in its conversations** (`chat/TheirMessages`, "What Ana said"), for every
  report: a report from a player's card names no message, and one line cannot be judged alone. `ListMessages` with
  the account's id reads its messages, 20 at a time (`THEIR_PAGE`), then `moreConversations`
  (`theirConversations.ts`) reads the whole chat around each, `AROUND` (5) messages each side, and shows them at
  once, newest first, three at a time (`CONVERSATIONS`), "Show more conversations" for the next. **One of their
  messages a conversation already holds gets none of its own, and a conversation that reaches the next one joins
  it**, so an exchange of three insults is one conversation, read with two calls. Their lines are marked
  (`line-theirs`) and tagged `reported`, the reporter's `reporter`. Each conversation grows with "Show earlier" and
  "Show later". Nothing kept is "Nothing they said in the chat is kept."
- **Each line** has its author's name as the chat names it now, its country and its time (with the date when it
  is not the reported message's day), what it answers ("In reply to …"), and is struck through with when it was
  removed. **What the reporter and the reported account said is tagged** `reporter` and `reported`, by account
  id. An author's name opens it on the player page (`OpenPlayer`), so a third player in the conversation is a
  click away; the player already open there stays open, with its row. A page past the chat's retention is
  `not_found`: "That part of the chat is no longer kept."
- **The actions, all on the reported account id**, never its name: remove the message (`RemoveMessage`, only while
  it is shown), mute (`Mute`, an hour by default), ban (`BanPlayer`, no duration by default, so the antibot's
  ladder picks the next step) and inspect (`InspectPlayer`), both `antibot.v1`'s. Then close the report (`DecideReport`): Acted or
  Dismissed, with an optional comment of up to 1000 characters. A second decision replaces the first.
- **"Every tool →" opens the reported account on the player page** (`OpenPlayer` in `navigation.ts`, which `App`
  provides): unban, revert, its address. The mute, ban and inspect here are the player page's own components.
- **`reports/actions.ts` holds the report's own calls** (remove, decide); the mute, ban and inspect are
  `players/actions.ts`'s.
