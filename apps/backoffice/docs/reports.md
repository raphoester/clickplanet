# Acting on a report

`src/reports/` is the Reports page, an inbox: `ReportsView` shows the list and the report open beside it from 1280px,
one or the other below. The report open is in the address bar (`#/reports/report-3`), so a link or a reload opens it.

- **The list reads every report once** (`ListReports`, `MAX_REPORTS`, the server's 500) and filters on the page: Open
  (the default) or All, each with its count. A row (`ReportList`) is the reason, the reported name, how long ago, and
  **the reported line itself** (`GetMessage` per open report, `ReportedText`), or what the reporter wrote; then who
  reported. An account reported more than once shows `×N`. A closed report shows its outcome and comment.
- **The report** (`ReportDetail`), top down as it is judged: the reported line quoted large (struck through once
  removed; "This message is no longer kept." past the chat's retention), what the reporter wrote, who reported whom
  (each name opens the player page, each short id has Copy), **where the reported account stands** (`Standing`, from
  `InspectPlayer` on open: banned until when, the jury's opinion, a pardon; and `GetMute`: muted until when), and its other reports, open or closed,
  each one click away.
- **The verdict** (`VerdictPanel`, `verdict.ts`) is one form under the header: remove the message, mute for, ban for
  (the ladder's next step by default), a comment, and "Close the N other open reports on Ana too" (on by default).
  **Act and close** (or "Close as acted" with nothing ticked, after a ban from the player page) and **Dismiss** each
  ask one question that names every step ("Remove the message, mute Ana for 1 day and close this report as Acted?").
  The steps run in order, then each report closes (`DecideReport`). **A step the server refuses stops the rest**: what
  was done is said, then the refusal, and the report stays open (`Halted`). A comment left empty records what was
  done ("Removed the message. Muted for 1 day."). A second decision replaces the first.
- **After a verdict the next open report opens** (the one below, or the first left), the list loses the ones closed,
  and a green line says what the verdict did. A report picked by hand clears it.
- **The chat around it** (`Conversation`): a message alone cannot be judged, so the page pages through the chat
  around it with `ListMessages`, the message's id as the cursor: the 10 messages before it and the 10 after
  (`FIRST_PAGE`), at once, then 20 more each time "Show earlier" or "Show later" is pressed (`NEXT_PAGE`), from
  the oldest or the newest message shown. It sits in a box of its own height that opens on the reported line, in its
  middle. A page shorter than asked is the end, and its button goes. The reported message shows while they load,
  marked ("Reported message", `aria-current`), and stays shown if they fail. `GetMessage` names no author, so its line
  takes the name the report gives its account.
- **What the reported account said, in its conversations** (`chat/TheirMessages`, "What Ana said"), folded
  (`components/Fold`): it reads only once opened, and opens by itself for a report that names no message (from a
  player's card), where it is the only evidence. `ListMessages` with the account's id reads its messages, 20 at a time
  (`THEIR_PAGE`), then `moreConversations` (`theirConversations.ts`) reads the whole chat around each, `AROUND` (5)
  messages each side, and shows them at once, newest first, three at a time (`CONVERSATIONS`), "Show more
  conversations" for the next. **One of their messages a conversation already holds gets none of its own, and a
  conversation that reaches the next one joins it**, so an exchange of three insults is one conversation, read with
  two calls. Their lines are marked (`line-theirs`) and tagged `reported`, the reporter's `reporter`. Each
  conversation grows with "Show earlier" and "Show later". Nothing kept is "Nothing they said in the chat is kept."
- **Each line** has its author's name as the chat names it now, its country and its time (with the date when it
  is not the reported message's day), what it answers ("In reply to …"), and is struck through with when it was
  removed. An author's name opens it on the player page (`OpenPlayer`), so a third player in the conversation is a
  click away; the player already open there stays open, with its row. A page past the chat's retention is
  `not_found`: "That part of the chat is no longer kept."
- **"What the antibot reads"** is the whole inspection, folded under the chat.
- **`reports/actions.ts` holds the report's own calls** (remove, decide); the mute and ban are `players/actions.ts`'s.
