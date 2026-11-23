# Answering a note

`src/support/` is the Ideas and bugs page, an inbox: `NotesView` shows the list and the note open beside it from
1280px, one or the other below. The note open is in the address bar (`#/notes/note-3`), so a link or a reload opens it.
The backend's side is [Answering a note](../../backend/docs/support.md#answering-a-note).

- **The list reads the notes twice, at once** (`ListNotes`, `MAX_NOTES`, the server's 500): the open ones
  (`open_only`: an address, and the player wrote last) and every note. Open (the default) or All, each with its count.
  All lists the open notes first. Each list comes the note written to last first (`latestFirst` in `notes.ts`). A row
  (`NoteList`) is the kind, who sent it, how long ago it was last written to, what the player wrote last (its last
  reply, or the note), and where it stands: New, Replied, Answered, Closed or No address (`statusOf`).
- **The note** (`NoteDetail`), top down: its kind and time, who sent it (see [Who sent a note](note-sender.md)), the
  address an answer goes to, with Copy, the device, folded, and Open player. Then **the conversation**
  (`conversationOf`): the note with its files (see [A note's photos and videos](note-media.md)), then the answers
  ("ClickPlanet"), the player's replies and the last close, by time. **A reply from another address than the note's
  shows that address** in place of the player's name.
- **The answer** (`AnswerPanel`): plain text up to 5,000 characters (`MAX_ANSWER_LENGTH`), its lines kept, its ends
  trimmed. **Send asks first and names the address** ("Send this answer to ada@example.com?"), then `AnswerNote` mails
  it. **Close without answering** asks too, then `CloseNote`: the note waits for nothing until the player writes again.
  Only an open note has Close; a note answered already can be answered again.
- **After a send or a close the next open note opens** (the one below, or else the first left), the list loses the one
  done, and a green line says what was done ("Ada: Sent to ada@example.com."). A note picked by hand clears it. Reload
  reads both lists again.
- **A refusal is said under its button, and the text typed stays** (`refusalOf` in `support/actions.ts`):
  `unavailable` is a mail that did not go, and nothing was kept ("The mail did not go. Send it again.");
  `failed_precondition` a note with no address; `not_found` a note no longer kept.
- **A note with no address cannot be answered** (sent before the notes kept one, or anonymized at 90 days): the page
  says so, and shows no answer box.
- **The Home page counts the open notes** (see [The home page](home.md)).
