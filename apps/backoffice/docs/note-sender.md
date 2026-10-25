# Who sent a note

**A note is sent from an account, always**: `SuggestFeature` and `ReportBug` refuse a caller with no session, and
`ListNotes` answers its `account_id`. It is blank only once the account is deleted (the backend anonymizes the
note and keeps its text).

- **"From" is the sender's name and account id, with Copy** (`Person`). `ListNotes` carries no name, so `NoteList`
  asks `player.v1.AdminService/GetAuthors` for the accounts of the page (`namesOf` in `players/actions.ts`, which
  the player list's `withAuthors` uses too). **Names that cannot be read leave the notes listed**, every sender
  "no name", with the reason above them.
- **"Open player" opens the sender on the player page** (`OpenPlayer`): its profile, its inspection, and from
  there the address of its latest take. A note with no account has no button.
