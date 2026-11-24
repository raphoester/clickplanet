# Flag submissions

**A signed-in player sends its region's flag for the operator's review, and reads the verdict in the game** (step 3 of
the regional flags, 2026-11-24): a name, a picture and its capital, a pin on the globe. The pin is the region's capital:
in Capital Siege the flag starts there, as a country starts on its capital. So the page says "capital" wherever the
player reads about the pin (the owner, 2026-11-24). The operator approves or rejects it in the backoffice; an approved
new flag waits to be opened (paid for), a change goes on its flag. The backend's side is `flags.v1.FlagService`'s
`CreateSubmissionUpload`, `SubmitFlag` and `ListMySubmissions` (see the contract in `proto/flags/v1/flags.proto`).

## Where

**At the end of the flag picker** (`FlagSubmissionEntry` under the list in `CountryPicker`, the desktop menu's and the
phone's Board sheet): it is where a player looks for its region's flag, and finds it missing. A `menu-tile` row: "Add a
flag", or, once the account has a submission, the latest one's name and its state as a chip (Waiting, Approved,
Rejected, Open). It shows even while the picker's list is locked.

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
  the flag a change is for. What was typed is in the store, so it stays while the form is aside.
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
  `refusal`): `no-session`, `guest`, `not-yours`, `waiting` (one already waits), `invalid` (said as the picture's, since
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
- **A change is prefilled** with the flag's capital, `Sold.pin` from the list of flags: coordinates alone, with no
  place name.

## The status, in the game

**The place shows the latest submission**: its name, its state's chip and one line. Waiting: "Sent on Nov 24, 2026."
Approved: "Approved! It opens soon.", or for a change "Approved! The change is on your flag." Rejected: its reason in
plain words (not a place, a flag in the game already, unreadable at a tile's size, hateful, or "It was not approved."),
then the operator's note quoted. Opened: "Bavaria is in the game!", the picture an approval made public beside the name.

- **A new submission can be sent once none waits**: "New flag" after an approval or an opening, "Try again" after a
  rejection, which opens the form with what was sent (the name, the capital, the flag of a change; the picture is
  picked again). While one waits, Close alone.
- **A change**: "Your flags" lists each flag the account founded (`foundedFlags`: one of its submissions opened it, or an
  approved change named it), each with Change, off while one waits. The form opens on the flag's name from the list of
  flags and its capital; Send waits for a picture, the same file for a new name alone: the server draws and reviews
  every flag as the players will see it.

**When the page reads them** (`useFlagSubmissionsFeed`): at once for a signed-in account, each time the flags stream
opens (`useFlags.openings`, which the flags store counts as it reads the list again), and when the place opens. One read
at a time, one queued behind it; a read that fails keeps what is shown. A guest reads none, and a sign-out forgets them.

**A decision is told once on the news line** (`SubmissionNews`, the news store's `submission` kind, 4 seconds): "Your
flag Bavaria is approved!", "Your change to Bavaria is approved!", "Your flag Bavaria was rejected.", "Bavaria is in the
game!". The first time the page sees a submission in a decided state, the newest one is told, and each one's state is
kept in `localStorage` under `clickplanet-flag-submissions-told` (`{id: state}`), so an approval then an opening is told
twice, and a reload tells nothing again. A browser that keeps nothing tells nothing.

## In fake mode

`FakeFlagSubmissions` (`backends/fakeFlagSubmissions.ts`) keeps the submissions in memory, uploads in eight steps so the
bar moves, and refuses as the server does (a guest, one waiting, a flag not founded). Sign in first (`signIn()`), send a
flag from the picker, then on the console:

- `approveSubmission()`: the one that waits is approved, with its upload as its public picture; a change goes on its
  flag at once.
- `rejectSubmission("not-a-place", "A club's crest.")`: rejected, with a reason (`not-a-place`, `exists`, `unreadable`,
  `hateful`, `other`) and a note.
- `openSubmission(days)`: the approved new flag opens (`FakeFlagsBackend.open` with the upload's picture and the pin),
  for 30 days by default.
- `refuseSubmissions("waiting")`: the next sends are refused with that refusal, or `"upload-failed"` for a `PUT` that
  fails; `refuseSubmissions()` lets them through again.

After each verdict the fake says the flags stream opened again (`FakeFlagsBackend.reopen`), so the page reads it at once
and the news line tells it, as the real page would at the stream's next opening or the place's next opening.

## Not built yet

A decision read while the page is open and the stream stays up: the server tells none on a stream, so the page learns it
at the next opening of the flags stream, of the place, or a load.
