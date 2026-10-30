# Support

What a player sends the operator: a feature suggestion or a bug report, from
the foot of Settings, and a report of another player, from that player's card.

- `backends/support.ts` — `FeatureSuggester` and `BugReporter`, the
  `PageContext` both send, `AttachmentRulesReader` and `AttachmentUploader` for
  the files (`FeedbackBackend` is those four together), `PlayerReporter` with
  its `PlayerReport` and `ReportReason` (and `SupportBackend`, every support
  contract together),
  `MAX_NOTE_LENGTH`, which mirrors the backend's
  `notes.MaxTextLength`, and the errors. `ConnectSupportBackend`
  (`supportBackend.ts`) implements it over `support.v1.SupportService`;
  `fakeSupportBackend.ts` is the dev stand-in.
- **A send carries the click token**, minted when none is held, as a chat
  message does: a guest sends too. A refused token is dropped and the call made
  once more with a fresh one; a second refusal, or a mint that failed, is
  `SupportNoSessionError`.
- **A send is never retried.** A call that landed and lost its answer would be
  kept twice, and the text is still in the box.
- The refusals are `resource_exhausted` → `SupportRateLimitedError` (one address
  sends three, then one a minute; an account and an address send ten notes and a
  hundred files a day; and the bucket is full) and `invalid_argument` →
  `SupportRejectedError`. The line says "Try again later": the wait can be a day.
  A player report has one more: `already_exists` → `SupportReportedAlreadyError`
  (one account reports one player once a day).
- **A note says what the page knows about itself, a suggestion as a bug report**
  (`app/support/pageContextOf`): the build, the viewport in CSS pixels and the
  pixel ratio, whether the connection was down, the cores and the memory the
  browser reports, and what WebGL says of the graphics card. `Viewer` hands
  `SettingsPlace` a `pageContext()` that is read when the note is sent, not when the
  form opens. The server reads the user agent off the request.
- **The graphics card comes from the globe's own WebGL context**
  (`viewer/graphicsReport.ts`, through `Globe.graphics()`): the unmasked vendor
  and renderer when the browser gives them, its own names otherwise, the WebGL
  version, the largest texture, and whether the context is lost. A page cannot
  read `chrome://gpu`; this is what it can read of it. No second context is
  made to ask: a phone has few to give.
- **The build is `BUILD`, the file the bundler put `pageContext.ts` in**
  (`buildOf(import.meta.url)`): `play-<hash>.js` in production, so it names the
  bundle with no build variable to set, and `pageContext.ts` in dev.
- `app/support/NoteModal.tsx` is the one form, for all three: a `Modal` with a text
  box and Send. It opens on its first control. The text keeps its lines. A counter shows from 100
  characters left. A failure is one line under the box (`sendFailure.ts`), and
  the text stays. Once sent it says so and offers Close. A question before the
  text is `asked`, and Send waits for `ready`; `optional` lets the text stay
  empty, and makes the box short.
- **`FeedbackModal` is the one entry for an idea and for a bug**, opened by the
  Feedback tile ("Send feedback" wrapped on a phone 360px wide): `NoteModal`
  with "It is" before the text, a `radiogroup` of An idea and A bug (`Choices`,
  which the player report draws its reasons with too). The pick names the box
  and decides the call: `suggestFeature` or `reportBug`. Send waits for the pick
  and for the text, and the text stays when the pick changes. Two tiles for the
  two calls were one tile too many in a place of eight.
- **A click beside it does not close it once something is typed**
  (`stayOnBackdropClick`): the ×, Escape and Close still do.
- `SettingsPlace` shows Feedback only with a `support` backend wired. It asks
  for `FeedbackBackend` (the two sends and the two contracts of the files), not
  the whole `SupportBackend`.
- **A player is reported from its card.** With a `reporter` wired, `PlayerCard`
  has a Report button in its footer, clear of the body's scroll fade. A press
  shows `ReportPlayerModal` **in place of the card**, not over it: two modals
  would both close on one Escape. Closing the report steps back to the card.
- **`ReportPlayerModal` is `NoteModal` with a question before the text**
  (`asked`): `Choices` of the four reasons, then the details, which are
  `optional`. Send waits for a reason (`ready`). It opens on the first reason.
  It hands the form no `files`, so a report of a player takes none.
- **The report names the account, never the name.** A name can change hands,
  so a report by name could land on another player. `PlayerLine.accountId` is on
  every line a card opens from: a chat message (`authorAccountId`), a roster
  entry and a standing carry it off the wire. `ReportPlayerModal` takes the
  `accountId` to send and the `name` for its title only. **A line with no
  account id has no Report**: the author of a message kept before accounts, or
  whose account is deleted.
- **The card's report names a player and no message**, wherever the card was
  opened from. Opened from a chat name it once sent that message's id, and the
  name is only on the first message of its author's run: the report of an
  insult sent after "hi" pointed to "hi".
- **Every chat message has a Report button of its own**, beside react and reply
  (`.chat-report-add`, shown like them). It opens `ReportMessageModal`, held by
  `ChatPanel`: `NoteModal` with the message quoted, its author and its text, and
  optional details. **No reason is asked**: the message is what is reported, so
  it is abuse in the chat; the reasons are the card's, about a player. It sends
  the message's own id. A message sent is a filled flag, kept in `ChatPanel`
  (`reported`) so a phone's sheet closed and opened again still shows it. A pill
  that armed on the first press and sent on the second was tried: it covered
  the reply and reaction buttons, and left no room for details. None on the
  player's own messages (`useChat`'s `own`) or on a message with no account;
  none without a `support` backend wired.
- **No Report on the player's own card**, told by account id: `AccountStore`
  keeps the `accountId` of a signed-in player from its profile. A guest's is
  not read here, so its own card has the button, and the server refuses the
  report.
- The fakes name one account per name (`backends/fakeAccounts.ts`), so a fake
  chat message, roster line and standing agree on who a player is.

**A note carries photos and videos**, an idea as a bug: `FeedbackModal` hands
`NoteModal` the backend as `files`, and the picker is under the text whatever
the pick. See the backend's [Support](../../backend/docs/support.md) for where they are kept.

- **The server says what a note may carry.** The form reads `attachmentRules()`
  when it opens (`GetAttachmentRules`, no token): the most files, the most bytes
  of one, and the content types. The most bytes shrink to the room the bucket has
  left, and a full bucket takes no file, so the picker goes away. The picker shows only once they say one file or
  more, so a server with no bucket, an older server (it answers
  `unimplemented`) and a read that failed all give the plain form. No limit is
  written on this side.
- **The files first, the note after.** Send uploads each file in turn, then
  sends the note with the id of every one (`attachmentIds`): a note never
  reaches the operator without the files that explain it. `uploadAttachment`
  is two calls: `CreateUpload` (the type and the size, with the click token)
  answers an id and a link signed for that type and that size, and the file is
  put there. The server checks each id is in the bucket before it keeps the
  note.
- **`putFile` is one `PUT` straight to the bucket**, over `XMLHttpRequest`:
  `fetch` says nothing while a body goes up, and a 50 MB video on a phone needs a
  bar. It sets the `Content-Type` the link was signed for; with another one the
  bucket refuses the file. It is not a Connect call, so it has no token, no
  retry and no `connectionInterceptor`.
- `domain/attachments.ts` — `withPicked`, what the rules take of the files just
  picked, and the first refusal as one line (`refusalLine`): too many, too big,
  not a photo or a video. `sizeLine` writes a size.
- `app/support/Attachments.tsx` — `AttachmentPicker` (the button, a hidden file
  input with the server's types as `accept`, the list with a thumbnail, the size
  and a remove button), each with its bar while it goes up and a tick once it
  is up. **A picture pasted into the text is a file too.** A photo the browser
  cannot draw (HEIC outside Safari) gets the plain mark.
- **The form shows each file going up.** The footer is "Uploading 2 of 3…", the
  text is read-only, the files cannot be taken off, and a click beside the form
  does not close it. A file that did not upload stops the send and is named in
  one line; Send again goes on from it. **A file already up is not sent again**
  (`NoteModal` keeps each one's id), on a failed upload or a refused note
  alike.
- **Closing the form calls the upload off and sends nothing** (an
  `AbortController` made in an effect). Made once per component, React's
  StrictMode aborted it before the form was ever used, and every upload in
  `npm run dev` stopped at 0%. A file already up stays in the bucket, named by
  no note.
- The fake uploads in eight steps, so the bar moves in dev.
