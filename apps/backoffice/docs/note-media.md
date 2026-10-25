# A note's photos and videos

`Attachments` shows each file a note carries, under its text, from the `url` `ListNotes` answers.

- **A photo is a thumbnail that opens the file**, a video plays in place (`preload="metadata"`), and every file has
  its type, its size and an Open link.
- **A file the browser cannot show** (HEIC outside Safari, a QuickTime video Chrome cannot decode) or a link past
  its hour says so, and keeps the Open link. The links are signed for an hour (`support.attachments.linkTTL`):
  Reload signs new ones.
