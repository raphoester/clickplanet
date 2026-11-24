# Flag submissions

**A signed-in player sends its region's flag for the operator's review, and reads the verdict in the game** (step 3 of
the regional flags, 2026-11-24): a name, a picture and its capital, a pin on the globe. The pin is the region's capital:
in Capital Siege the flag starts there, as a country starts on its capital. So the page says "capital" wherever the
player reads about the pin (the owner, 2026-11-24). The operator approves or rejects it in the backoffice; an approved
flag waits to be opened (paid for). **The player changes it while it waits, never once the operator decided it** (the
owner, 2026-11-24: a "Change" on an approved flag "shouldn't be allowed"). The backend's side is `flags.v1.FlagService`'s
`CreateSubmissionUpload`, `SubmitFlag` and `ListMySubmissions` (see the contract in `proto/flags/v1/flags.proto`).

## Where

**At the end of the flag picker** (`FlagSubmissionEntry` under the list in `CountryPicker`, the desktop menu's and the
phone's Board sheet): it is where a player looks for its region's flag, and finds it missing. A `menu-tile` row: "Add a
flag", or, once the account has a submission, the latest one's name and its state as a chip (Waiting, Approved,
Rejected, Open, Closed: see [What the player reads](#what-the-player-reads)). It shows even while the picker's list is
locked.

- **Who sees it** is `useFlagSubmissions.who`, which `PlayScreen` hands `useFlagSubmissionsFeed`: `player` for a
  signed-in (linked) account, `guest` for a guest the server offers sign-in to, `nobody` otherwise (the account not read,
  or no provider offered), and then no entry.
- **A guest sees the sign-in buttons**, never the form: "Sign in to add your flag." over `SignInButtons` (Google,
  Discord, the emailed code), in the same modal. Signed in by email there, the place reads the account's submissions
  and shows the form.

## The place

**`FlagSubmissionPlace` is mounted with the game** (`PlayScreen`, once the globe is ready), not inside the picker: the
form is put aside while the pin is aimed, and on a phone the sheet that held the picker closes then. It reads two
stores:

- **`useFlagForm`** (`flagFormStore.ts`): whether the place is open (`shown`), the form rather than the status
  (`composing`), the pin aimed (`aiming`), and the `Draft`: the name, the `File`, the pin with the place's name, and
  the submission that waits, which a change replaces. What was typed is in the store, so it stays while the form is
  aside.
- **`useFlagSubmissions`** (`submissionsStore.ts`): the account's submissions, the newest first, and whether a read
  ended.

It shows the form while composing or when the account has none, the latest submission's status otherwise ("Loading…"
until the first read), and a guest's sign-in. Opening it reads the submissions again.

## The form

`FlagSubmissionForm`, a `Modal` like the support's `NoteModal`:

- **The name**: 1 to 40 characters once the spaces at its ends are cut, counted as code points (`flagNameLength`, the
  server's `catalog.ParseName`); "7/40" beside its label, red past 40.
- **The picture**: one file, shown as picked. Its type (PNG, JPEG, WebP) and size (5 MB) are checked as it is picked
  (`pictureFileRefusal`), and its shape once the preview loads (`pictureShapeRefusal`: between square and 2 wide by 1
  high, 100 pixels wide at least): a refused file is not kept, and one line says why. The server checks it all again,
  and refuses a moving picture, which the page cannot tell.
- **The capital**: "Put the capital on the globe" puts the form aside and aims the globe (below); once put, the name of
  the place under the pin and its coordinates ("Upper Bavaria", "48.14° N, 11.58° E", each language's compass letters),
  and "Move".
- **Send**: `CreateSubmissionUpload` with the type and the size, the `PUT` to the link with a bar on the picture and
  "Uploading… 50%" on the button, then `SubmitFlag`. Its answer goes first in the store (`submitted`) and the place shows
  its status. **Each send uploads the picture again**: the server may have used or dropped an upload it refused.
- **Each refusal is one line under the form, and the form keeps what was typed** (`SubmissionRefusedError`'s
  `refusal`): `no-session`, `guest`, `decided` (the operator decided the one a change replaces: "Your flag was reviewed
  already, so it can't change."), `waiting` (one already waits), `invalid` (said as the picture's, since
  the page checked the rest), `off` (pictures off on the server), `too-many` (the uploads of a day); a `PUT` that failed
  is "The picture did not upload".
- **Closing it calls the upload off** (an `AbortController` made in an effect, as `NoteModal`'s: made once per component,
  StrictMode aborted it before any use). A click beside it does not close it once something is typed.

## Aiming the capital

**As the rally is aimed** (see [Rally](rally.md)): `useFlagForm.aim()` sets `aiming`, `useGlobe` follows the store and
calls `Globe.setPinAimed(true)`, which puts every bonus aim away and sets the crosshair (`viewer-canvas--pinning`).

- **A tap on any point of the globe**, land or sea, at any zoom (no fly down from orbit), is the pin: `pinAt` reads the
  point as the tile positions place it (y the north pole, longitude 0 on -x, the backend's convention), and the place
  named there is the province's under the tap, or the landmass's, or none at sea. The globe tells the pin
  (`onPinned`), then puts the aim away; the store takes a pin only while it is aimed.
- **Escape, Cancel, or a bonus aimed** puts it away (`onPinAimChange(false)`), and the form comes back as it was.
- **While aimed, a line over the globe** says "Tap your region's capital", with Cancel (`PinAim`, at the news line's
  place).
- **A change is prefilled** with the capital that was sent: coordinates alone, with no place name.

## The status, in the game

**The place shows the latest submission**: its name, its state's chip and one line. Waiting: "Sent on Nov 24, 2026."
Approved: "Approved! It opens soon." Rejected: its reason in
plain words (not a place, a flag in the game already, unreadable at a tile's size, hateful, or "It was not approved."),
then the operator's note quoted. Opened: "Bavaria is in the game until Dec 24, 2026!" (its `endsAt`; "Bavaria is in the
game!" while the list of flags does not hold it yet), the picture an approval made public beside the name. Closed:
"Bavaria closed on Nov 24, 2026." (its `closedAt`), then the operator's note quoted when the operator closed it (below).

### What the player reads

**Once a submission opened its flag, the page shows the flag's state now, not the submission's** (`shownOf` in
`domain/flagSubmissions.ts`): the server keeps the submission `opened` for good, and the flag lives on in the flags
module, which closes it at its end, at the operator's `CloseFlag`, and opens it again at an `AddTime` within 30 days. So
an opened submission whose flag the list of flags holds closed reads Closed, a muted chip, in the picker's entry and
the place, and Open again once its flag comes back. "Your flags" marks each closed flag with the same chip.

**Why the operator closed it, in its own words** (`ListMySubmissionsResponse.closures`, a note per flag the account
founded that the operator closed and that has not come back; see the backend's [flags module](../../backend/docs/flags.md)):
`ConnectFlagSubmissions` puts each note on the submission that opened its flag (`Submission.closeNote`), so the store
holds submissions alone. The place quotes it under "Bavaria closed on …", as a rejection's note, while the flag reads
Closed; and on that flag's "Your flags" row when the card shows another submission. A flag closed at its end, or
before the notes existed, has none: the date alone. **The page reads the submissions again when a flag the account
founded closes or comes back** (`useFlagSubmissionsFeed`, on the list of flags), so the note shows at once in a place
left open. The note is the operator's, in English, never translated.

- **A new submission can be sent once none waits**: "New flag" after an approval or an opening, "Try again" after a
  rejection, which opens the form with what was sent (the name and the capital; the picture is picked again). The
  footer holds that one next step, and the × closes.
- **A change, while it waits alone**: Change, the footer's button while one waits (`waitingOf`), opens "Change your
  flag" with what was sent, and `replaces` its id. Send waits for a picture, the same file for a new name alone: the page keeps none, and the server
  draws and reviews every flag as the players will see it. The new one takes the old one's place, which the server
  deletes unreviewed. "Your flags" lists each flag the account founded (`foundedFlags`: one of its submissions opened
  it), with no Change: nothing changes a flag once the operator approved it.

**When the page reads them** (`useFlagSubmissionsFeed`): at once for a signed-in account, each time the flags stream
opens (`useFlags.openings`, which the flags store counts as it reads the list again), and when the place opens. One read
at a time, one queued behind it; a read that fails keeps what is shown. A guest reads none, and a sign-out forgets them.

**A decision is told once on the news line** (`SubmissionNews`, the news store's `submission` kind, 4 seconds): "Your
flag Bavaria is approved!", "Your flag Bavaria was rejected.", "Bavaria is in the
game!", "Your flag Bavaria is closed.". The first time the page sees a submission in a decided state, the newest one is
told, and each one's state as the player reads it is kept in `localStorage` under `clickplanet-flag-submissions-told`
(`{id: state}`, `closed` among them), so an approval, an opening then a close is told thrice, a comeback again, and a
reload tells nothing again. A browser that keeps nothing tells nothing. **Nothing is told before the list of flags is
read** (`useFlags.read`): an opened flag would be told open, then closed. The page looks again when the submissions or
the list of flags change, so a close the flags stream tells is told at once.

## In fake mode

`FakeFlagSubmissions` (`backends/fakeFlagSubmissions.ts`) keeps the submissions in memory, uploads in eight steps so the
bar moves, and refuses as the server does (a guest, one waiting, a change of one decided). Sign in first (`signIn()`), send a
flag from the picker, then on the console:

- `approveSubmission()`: the one that waits is approved, with its upload as its public picture.
- `rejectSubmission("not-a-place", "A club's crest.")`: rejected, with a reason (`not-a-place`, `exists`, `unreadable`,
  `hateful`, `other`) and a note.
- `openSubmission(days)`: the approved flag opens (`FakeFlagsBackend.open` with the upload's picture and the pin),
  for 30 days by default.
- `refuseSubmissions("waiting")`: the next sends are refused with that refusal, or `"upload-failed"` for a `PUT` that
  fails; `refuseSubmissions()` lets them through again.

After each verdict the fake says the flags stream opened again (`FakeFlagsBackend.reopen`), so the page reads it at once
and the news line tells it, as the real page would at the stream's next opening or the place's next opening. Then
`closeFlag(code, note)` (the code `openSubmission` answers; a note by default) closes it as the operator would, with the
note its founder reads, and `addTime(code, 7)` brings it back.

## Not built yet

A decision read while the page is open and the stream stays up: the server tells none on a stream, so the page learns it
at the next opening of the flags stream, of the place, or a load.
