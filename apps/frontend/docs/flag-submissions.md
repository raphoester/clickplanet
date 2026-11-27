# Flag submissions

**A signed-in player makes its region's flag, sees it as the players will, sends it, then pays for it while it is
reviewed** (step 3 of the regional flags, 2026-11-24; one flow from every door, the owner, 2026-11-26): a name, a
picture and its capital, a pin on the globe. The pin is the region's capital: in Capital Siege the flag starts there,
as a country starts on its capital. So the page says "capital" wherever the player reads about the pin (the owner,
2026-11-24). **Commitment first, then the paywall**: step 1 makes the flag with no price, Next sends it, and the
server's Claude judge reviews it while the player is on step 2, the paywall (the backend's
[The judge](../../backend/docs/flags.md#the-judge): every 3 seconds, paid or not). An approved flag paid opens at once.
**The player changes it while it waits and is not paid, never once paid or decided** (the owner, 2026-11-24: a
"Change" on an approved flag "shouldn't be allowed"). The backend's side is `flags.v1.FlagService`'s
`CreateSubmissionUpload`, `SubmitFlag` and `ListMySubmissions` (see the contract in `proto/flags/v1/flags.proto`); the
payment is the shop's (see [The shop](shop.md)). **A flag lives while someone pays for it** (the owner, 2026-11-26):
its holder adds time, and once its time ends any player may claim it (the backend's
[Holders and claims](../../backend/docs/flags.md#holders-and-claims)).

## The doors

**Every door opens the same flow**, once the account's submissions are read ("Loading…" until the first read):

- **The shop's** (`useFlagForm.openShop`): the phone's status-bar bag and the desktop rail's Shop, while the shop sells
  (see [The shop](shop.md#its-doors)). It opens [Your flags](#your-flags) when the account holds a flag or sent one
  that did not open, and step 1 when it has none (the owner, 2026-11-26: the door opened the account's abandoned,
  approved Sardinia on its paywall, with no way to another flag).
- **The flag picker's row** (`FlagSubmissionEntry`, at the end of `CountryPicker`'s list, the desktop menu's and the
  phone's Board sheet): always "Add your region's flag", gold with the shop door's shine (`shop-shine`, none under
  reduced motion), and step 1 (`useFlagForm.open`). It never shows a submission: until 2026-11-26 it showed the latest
  one ("Bavaria · Closed"), and a founder lost its way to a new flag. It shows even while the picker's list is locked.
- **The picker's search that finds no country** (`SelectWithSearch`'s `noMatch`; the selected country, which the list
  always keeps, does not count): the row comes up under the search, larger, "Put “Bavaria” on the map", in place of
  the one under the list, and the text typed is the flag's name on step 1 (cut at its ends). **When the search finds
  a flag whose time ended** (`FlagCatalog.claimableCodes`, matched as the picker matches a name), the row is "Bavaria is
  free. Claim it." and opens [its pay page](#a-flags-pay-page) as a claim (`useFlagForm.payFor`). A removed flag is never
  offered: its name is free for a new flag.
- **Who sees the row** is `useFlagSubmissions.who`, which `PlayScreen` hands `useFlagSubmissionsFeed`: `player` for a
  signed-in (linked) account, `guest` for a guest the server offers sign-in to, `nobody` otherwise (the account not
  read, or no provider offered), and then no row.
- **A guest sees the sign-in buttons**, never the form: the region's flag's pitch, "Sign in to add your flag." over
  `SignInButtons` (Google, Discord, the emailed code), in the same modal. Signed in by email there, the door lands once
  the submissions are read.

## The place

**`FlagSubmissionPlace` is mounted with the game** (`PlayScreen`, once the globe is ready), not inside the picker: the
form is put aside while the pin is aimed, and on a phone the sheet that held the picker closes then. It reads two
stores:

- **`useFlagForm`** (`flagFormStore.ts`): whether the flow is open (`shown`), its `step` (`door` until the submissions
  are read, `hub` for Your flags, `make`, `pay` with the submission as sent and what was typed for it, or `flag` for a
  flag's pay page), the pin aimed (`aiming`), and the `Draft`: the name, the `File`, the pin with the place's name, and
  the submission that waits, which a change or a new flag replaces. What was typed stays while the form is aside, and
  on step 2 for Back and Fix it.
- **`useFlagSubmissions`** (`submissionsStore.ts`): the account's submissions, the newest first, the flags it holds
  (`held`, each removed one with its note), and whether a read ended.

Opening it reads the submissions and the shop's plans again.

## Step 1: make your flag

`FlagSubmissionForm`, a `Modal` titled "Add your region's flag", or "Change your flag" for a change. **No price shows.**

- **The head** (`FlagPitch`), over a new flag only: the shop's gold coin, "Put your region on the map" and its three
  perks.
- **A new flag never dead-ends.** The server keeps one submission waiting at a time, so a new flag takes the place of
  the one that waits unpaid (`replacedByNewOf`, `Draft.replaces` and `Draft.replacing`), and says so under the head:
  "This replaces Sardinia, which waits." An approved one not paid does not block a new flag. If the judge decides the
  one it replaces while the player types, the send is refused `decided` and sent again at once with nothing replaced.
  A paid one that waits is not replaced: the server refuses the send (`waiting`), and it is reviewed within seconds.
- **A place has one flag**: a name that is a flag whose time ended (`FlagCatalog.claimableNamed`, whatever its case)
  shows "Bavaria is free. Claim it." under the name with Claim it, its pay page, and Next stays off.
- **The name**: 1 to 40 characters once the spaces at its ends are cut, counted as code points (`flagNameLength`, the
  server's `catalog.ParseName`); "7/40" beside its label, red past 40.
- **The picture**: one file, shown as picked. Its type (PNG, JPEG, WebP) and size (5 MB) are checked as it is picked
  (`pictureFileRefusal`), and its shape once the preview loads (`pictureShapeRefusal`: between square and 2 wide by 1
  high, 100 pixels wide at least): a refused file is not kept, and one line says why. The server checks it all again,
  and refuses a moving picture, which the page cannot tell.
- **The capital**: "Put the capital on the globe" puts the form aside and aims the globe (below); once put, the name of
  the place under the pin and its coordinates ("Upper Bavaria", "48.14° N, 11.58° E", each language's compass letters),
  and "Move".
- **The flag card** (below), the flag on the game's map at its capital, under the fields.
- **Next**: `CreateSubmissionUpload` with the type and the size, the `PUT` to the link with a bar on the picture and
  "Uploading… 50%" on the button, then `SubmitFlag` (with `replaces` for a change), and no checkout. Its answer goes
  first in the store, in the place of the one it replaced (`submitted`), and step 2 shows it. **Each send uploads the
  picture again**: the server may have used or dropped an upload it refused.
- **Each refusal is one line under the form, and the form keeps what was typed** (`SubmissionRefusedError`'s
  `refusal`): `no-session`, `guest`, `decided` ("Your flag was reviewed already, so it can't change."), `waiting` (one
  already waits), `invalid` (said as the picture's, since the page checked the rest), `off` (pictures off on the
  server), `too-many` (the uploads of a day); a `PUT` that failed is "The picture did not upload". **A change refused
  `decided` whose flag the store holds shows that flag's paywall**, read again: the judge decided it while the player
  changed it, and its verdict is there; Fix it after a rejection brings back the change typed.
- **Closing it calls the upload off** (an `AbortController` made in an effect, as `NoteModal`'s: made once per component,
  StrictMode aborted it before any use). A click beside it does not close it once something is typed.

### The flag card

**`FlagCard` shows the flag on the game's own map, at its capital** (the owner, 2026-11-26: "a real screenshot of the map",
"centered on the capital location", in place of a few hexagons drawn flat on the card). Once the form has a picture and a
capital, `useMapShot` asks the globe for a shot (`Globe.previewFlag`), the name and the capital's place under it.

- **The globe flies to the capital** at the zoom a fly-down plays at (`flownZoomsOf`'s most, a tile 28 CSS px), behind
  the modal, and stays there. Once the flight ended and the picture is in the atlas, the capital is the tile under the
  pin, and it and 3 rings of tiles around it (`groundAround`, about 37 on land) wear the picture for one frame, under a
  flag code no flag has (`preview-1`, one per picture: each takes a sold flag's number for the page's life), **with no
  shield**: the badges of the shields players put there covered the new flag (the owner, 2026-11-26, at Bavaria). The
  frame is read (`captureRequests`, as the share's photo) and the tiles go back to what they show at once (`repaint`,
  their shields from `shielded`): paint only, as a ghost, so no count, version or read sees them. A pin at sea shows
  the sea.
- **The shot is the drawing buffer cropped around the capital** (`flagShooter`, `cropAround`): 560 x 280 CSS px, the
  widest card, at most the screen. The card shows it at that size, centred in a box of 2 wide by 1 high, so a tile on
  the card is a tile's size in the game on every screen; the crown is a DOM crown on the capital's pixel.
- **One shot per picture and capital for the visit** (`useMapShot`): the paywall shows the one the form took, a
  capital moved shoots again, and a shot asked for while another runs calls the first off (`AbortError`, no warning).
  An approved flag found by a door after a reload is shot from its public picture and its pin.
- **Until the shot**, and when it fails (no globe, a hidden tab), the box holds the picture alone; before a picture,
  or for a flag found after a reload with none public yet, there is no box.

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

## Step 2: the paywall

**`FlagPaywall` shows the flag sent, its review's verdict as it comes, and the plans once**: the flag card with its
state's chip, one line, and while the shop sells and the flag is not paid, `PlanOffer`: the plans, the consent and
Pay (`CreateCheckout{plan, submission_id, starts_now}`, see [The shop](shop.md#the-offer)), at the sale's prices with
its clock while a sale runs (see [The shop](shop.md#a-sale)). **Pay never waits for the verdict.** While the flag waits, the page reads `ListMySubmissions` every 3 seconds for 2 minutes at most
(`useVerdictReads`), and a newer read of the flag wins over the one sent.

- **Waiting, not paid**: "We check it after you pay." (the judge left it unsure, or has not answered yet; the operator
  reviews only the paid ones while the shop sells). "Sent on Nov 24, 2026." while the shop is off.
- **Approved, not paid**: the Approved chip, its public picture, "Approved! Pick a plan to open it.": paying opens it at
  once. "Approved! It opens soon." while the shop is off. The plan and the consent picked stay as the verdict lands.
- **Rejected**: its reason in plain words (not a place, a flag in the game already, unreadable at a tile's size,
  hateful, or "It was not approved."), then the operator's note quoted, then "Think we got it wrong? Ask a person to
  look again.", an email to support@clickplanet.lol named after the flag: the first review may be Claude's. **No plan
  shows, before any payment.** "Fix it" goes back to step 1 with the name, the capital and the picture typed: a
  decided submission is not replaced, so it sends a new one.
- **Paid**: "Paid! We check your flag, then it opens.", "Approved! It opens now." the moment before the flags stream
  tells it open, then "Bavaria is in the game!": no plan and no Back.
- **Back, while it waits and is not paid**: step 1 as a change of it (`replaces`), with what was typed. Beside Pay, or
  the footer's while the shop is off. Once approved or paid, none.
- **The picture typed stays for the visit**: a door that finds the flag after a reload has its name and its capital,
  and the picture is picked again for a change: the page keeps none.

## Your flags

**The shop's door opens `FlagHub`, "Your flags"** (the owner, 2026-11-26: "Your flags" left the profile, the desktop
`ProfileModal` and the phone You sheet, where it had moved that morning): a list of one line per flag the account
holds, then per submission whose flag did not open, and "Add a new flag" under it, the one red button (step 1, in the
place of the one that waits unpaid). It reads the submissions and the renewals again as it shows. 480 px wide on a
desktop, so a row's line reads on one line.

**A row is one button to the flag's own page** (`flag-row`, a box per row): the flag's picture (a grey flag for a
submission not yet approved, whose picture the server has not made public), its name over one line that says where it
is, and a chevron. **No button, no plans, no note in the list**: each is on the page the row opens (the owner,
2026-11-26: the first list put a chip, a line and a lone button on three corners of each row, "extremely slop"; the
second put a big button in each box, "monstrosities").

- **A flag held** (`ListMySubmissionsResponse.held`, the account's payment opened it or claimed it last): its time line
  (`flagTimeOf`, see [The shop](shop.md#time-for-a-flag-held)) while the shop sells, a removal's at all times; it opens
  [its pay page](#a-flags-pay-page). A flag another player claimed is no longer held: it leaves the list.
- **A flag sent**: the step 2 line; it opens its paywall (Pay, the verdict, Fix it, the note and the appeal).
- **What shows** (`flagRowsOf`): each flag held, then each flag sent whose flag did not open, **the newest of a name**
  (a second try of Kurdistan hides the first), and **none of a name the account holds** (a Bavaria rejected as `exists`
  while the account holds Bavaria).
- **A row that is done is put away** with its × ("Hide Sardinia"; `dismissable`), left of the chevron, which stays in
  line with the others: a flag removed, a flag rejected, an approved one left unpaid. Never one that waits. The keys
  are kept in this browser (`localStorage` `clickplanet-flag-rows-dismissed`, `useDismissedFlagRows`); a removed flag
  that comes back shows again. With every row put away, the shop's door opens step 1.

## A flag's pay page

**A flag held and a claim open `FlagPayScreen`, the same page as a new flag's paywall**, a modal: the flag card with
the game's map at its capital (`FlagCard`, `useMapShot` from the flag's public picture and pin), then the plans
(`PlanOffer`, target the flag). For a flag the account holds, its time line in place of the chip, and titled "Add time"
with the plans, or "Your flag" with none: a renewal that runs (Stop renewal over Back, side by side where the words
fit) or a removed flag (the operator's note, and Back). "Claim Bavaria" with the chip for one it does not hold. Back
goes to Your flags when it came from there. **Nothing is offered** for a removed flag, nor for an open one another
account holds (a claim that came too late). The server refuses a claim while another player's
renewal of the flag still runs ("You can't pay for this flag now.", see [The shop](shop.md#the-offer)).

### What the player reads

**Once a submission opened its flag, the page shows the flag's state now, not the submission's** (`shownOf` in
`domain/flagSubmissions.ts`): the server keeps the submission `opened` for good, and the flag lives on in the flags
module, which closes it at its end, removes it at the operator's `CloseFlag`, and opens it again for whoever pays. So
an opened submission whose flag the list of flags holds closed reads Closed.

**Why the operator removed it, in its own words** (`ListMySubmissionsResponse.closures`, a note per flag the account
holds that the operator removed; see the backend's [flags module](../../backend/docs/flags.md)): `ConnectFlagSubmissions`
puts each note on its flag held (`Held.removalNote`). The flag's pay page quotes it. **The page reads the
submissions again when a flag the account holds closes or comes back** (`useFlagSubmissionsFeed`, on the list of
flags), so the note shows at once and a flag claimed by another leaves the list. The note is the operator's, in
English, never translated.

**When the page reads them** (`useFlagSubmissionsFeed`): at once for a signed-in account, each time the flags stream
opens (`useFlags.openings`, which the flags store counts as it reads the list again), when the flow opens, every 3
seconds while the paywall shows a flag that waits, and every 3 seconds for a while after a payment (see
[The shop](shop.md#back-from-stripe)). One read at a time, one queued behind it; a read that fails keeps what is shown.
A guest reads none, and a sign-out forgets them.

**A decision is told once on the news line** (`SubmissionNews`, the news store's `submission` kind, 4 seconds): "Your
flag Bavaria is approved!", "Your flag Bavaria was rejected.", "Bavaria is in the
game!", "Your flag Bavaria is closed.". The first time the page sees a submission in a decided state, the newest one is
told, and each one's state as the player reads it is kept in `localStorage` under `clickplanet-flag-submissions-told`
(`{id: state}`, `closed` among them), so an approval, an opening then a close is told thrice, a comeback again, and a
reload tells nothing again. A browser that keeps nothing tells nothing. **Nothing is told of a flag another player
claimed since** (`untold` skips an opened submission whose flag the account no longer holds). **Nothing is told before
the list of flags is read** (`useFlags.read`): an opened flag would be told open, then closed. The page looks again when the submissions or
the list of flags change, so a close the flags stream tells is told at once.

## In fake mode

`FakeFlagSubmissions` (`backends/fakeFlagSubmissions.ts`) keeps the submissions in memory, uploads in eight steps so the
bar moves, and refuses as the server does (a guest, one waiting, a change of one decided or paid). Sign in first
(`signIn()`), make a flag from any door and press Next, then on the console, while the paywall shows it (it reads the
verdict within 3 seconds, as it would the judge's):

- `approveSubmission()`: the one that waits is approved, with its upload as its public picture; a paid one opens at once
  with its time.
- `rejectSubmission("not-a-place", "A club's crest.")`: rejected, with a reason (`not-a-place`, `exists`, `unreadable`,
  `hateful`, `other`) and a note; a paid one is refunded.
- `openSubmission(days)`: the approved flag opens (`FakeFlagsBackend.open` with the upload's picture and the pin),
  for 30 days by default.
- `refuseSubmissions("waiting")`: the next sends are refused with that refusal, or `"upload-failed"` for a `PUT` that
  fails; `refuseSubmissions()` lets them through again.

The fake shop pays for the flag on the paywall (see [The shop](shop.md#in-fake-mode)): `openSubmission()` is the opening
of an approved one without the plans.

After each verdict the fake says the flags stream opened again (`FakeFlagsBackend.reopen`), so the page reads it at once
and the news line tells it, as the real page would at the stream's next opening. The fake keeps which flags the page's
account holds: the ones its submissions opened and the ones it claimed. Then, with the code `openSubmission` answers:

- `endFlag(code)`: its time ends now. It reads Closed in Your flags, with Add time, and any player may claim it: the
  picker's search for its name offers "Bavaria is free. Claim it.". `endFlag("xb")` frees Brittany, which nobody holds.
- `claimElsewhere(code)`: another player claims a flag whose time ended: it opens again and leaves Your flags.
- `renewingElsewhere(code)`: another player's renewal still runs on it, so a claim is refused;
  `renewingElsewhere(code, false)` lets it through.
- `removeFlag(code, note)`: the operator's removal (a note by default), quoted on the flag's pay page; `addTime(code, 7)` brings it
  back.

## Not built yet

- **A decision read while no paywall shows and the stream stays up**: the server tells none on a stream, so the page
  learns it at the next opening of the flags stream, of the flow or the profile, or a load.
