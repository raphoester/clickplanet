# Flags: the sold flags

`src/flags/`, one page (`FlagsPage`), in the menu under Ops, at `#/flags`. **A sold flag is a real region's flag the
game does not ship with** (Brittany, England, Scotland and Wales shipped with it until 2026-11-24 and are sold since,
under their codes `xb`, `gb-eng`, `gb-sct`, `gb-wls`), opened for
every player until its end (step 2 of the regional flags: the owner opens one by hand, for free). The flags players
submit are judged on another page (see [Flag submissions](flag-submissions.md)). `flags.v1.AdminService` keeps them
(`ListFlags`, `CreateFlagUpload`, `OpenFlag`, `AddTime`, `CloseFlag`, `RenameFlag`; the backend's flags module). The
game's own flags are not on this page: they never end.

## Opening one

- **The form** (`OpenPanel`): a name of 1 to 40 characters, counted as it is typed; the end, a `datetime-local` in the
  operator's own time zone, which the page names, 30 days from now at first, with the days left beside it; **the
  capital**, a latitude and a longitude in degrees (where the flag starts in Capital Siege: the place's capital, or its
  middle; `pin` in the code and on the wire), with a link that opens the globe there; and one picture (`image/png`, `image/jpeg` or `image/webp`, 5 MB at most),
  shown as picked. A latitude and a longitude pasted together in either field ("48.1374, 11.5755", as a map copies
  them) go each to its own field (`pastedPlaceOf` in `pin.ts`). What the server would refuse is in red as it is typed
  (a name too long, another type, a file too big, an end gone by, a latitude past −90 to 90, a longitude past −180 to
  180), and **Open** stays disabled until the four are there and none is refused.
- **Open asks first**, and the question says what it costs: every player sees the flag at once and may play for it,
  and a flag is never deleted: closed, it stays in the list for good. Only Yes uploads anything.
- **Then three steps** (`openFlag` in `actions.ts`): `CreateFlagUpload` with the file's type and size, which answers an
  upload id and a signed link; one `PUT` of the file to that link with the same `Content-Type` (`putFile`, as a
  video's, see [Videos](videos.md)), with a progress bar; then `OpenFlag` with the name, the upload id, the end and the capital. The
  server draws the picture again and makes it public. The page says "Opened Rennes (r-…), until …", empties the form
  and reads the list again.
- **A refused step says why**: `CreateFlagUpload`'s and `OpenFlag`'s `failed_precondition` is "The flags' pictures are
  off on the server"; `invalid_argument` (a picture that moves, narrower than square or wider than 2 by 1, a name too
  long or another flag's, built-in or sold, whatever its case, an end) quotes the server; a failed `PUT` quotes the
  bucket. A try again sends the picture again: an upload link is cheap.

## The list

- **Every sold flag** (`ListFlags`, the game's own left out), each a card: its picture as the server drew it, its
  name, its code with Copy, its capital (`Sold.pin`, a link that opens the globe there), and Open, Ended or Removed.
  **An open flag** shows its end and the days left (to the nearest day, hours in its last two). **An ended one** (its
  time ran out, or a refund took it back) shows when it ended, and that any player may pay for it and hold it: a claim.
  **A removed one** (`Sold.removed`) shows when it was removed, and that nobody may pay for it and its name is free.
- **By end** (open first, the soonest to end first, then the closed ones, the last closed first) or **By name**. The
  contract has no time a flag opened, so there is no newest first.
- **Each read also names the flags to the rest of the console** (`learnFlags`, see
  [A country on the page](countries.md)): a flag opened here shows its picture and name on the other pages at once.
- **The list as last read stays while it is read again**, so each card keeps what the server answered it.

## Days and removing

- **Add days** (a whole number from 1 to 3660, 30 at first) asks first. For an open flag: "Add 10 days to Bavaria? It
  ends on … instead of …". For one ended or removed, however long ago: "Open Savoy again for 30 days, until …?", and
  every player sees it again with the land it still holds and its holder. Then it says the new end ("Open again. It
  ends on …") and reads the list again. Its `failed_precondition`: a removed flag whose name a new flag took since,
  "rename one of them first".
- **Remove** (the server's `CloseFlag`, for moderation: the EU's rules for online platforms ask for its note), on an
  open or an ended flag, needs a **note to its holder** first ("Note to its holder · 0/500", the server's
  `CloseFlag.note`, 1 to 500 characters once the spaces at its ends are cut; past 500 it says so in red): the holder
  reads it in the game, and nobody else does. A removed card keeps Remove greyed. It asks first: for good, its players
  must pick another flag, nobody may click or pay for it, its land stays, its name is free for a new flag; in a battle
  that locks the flags it plays on until the lock lifts; its holder reads your note; only Add days brings it back. Then
  it says "Removed." and reads the list again. Its `failed_precondition` is "It is removed already".

## Renaming

- **Rename**, on every card, open or closed: the name in a field, counted as it is typed as in the form (1 to 40
  characters, too long in red), and Rename disabled until it differs from the flag's own. It asks first ("Rename Bavaria
  to Bayern?", every player sees the new name at once), then says "Renamed to Bayern." and reads the list again. Its
  `invalid_argument` quotes the server: a name another flag has, built-in or sold, whatever its case, is refused there,
  not on the page. Its code, picture and land stay.

## Hosts

The upload goes to the R2 endpoint already in `connect-src`, and the pictures load from `https://media.clickplanet.lol`,
already in `img-src` (see [Deploy](deploy.md)): nothing to change in `public/_headers`.

## In the demo

`npm run demo`: Bavaria and Cornwall open (Cornwall in its last day), Savoy ended 3 days ago, Tyrol removed 41 days
ago, each picture drawn in the page (SVG), each capital where its place's is, and a few made-up players for Bavaria, so the
other pages show its picture. The made-up bucket takes a picture in two seconds; `OpenFlag` refuses a picture it does
not hold, a name of more than 40 characters or another flag's, an end gone by and a capital off the globe, and the flag
shows its picture from the page's memory until a reload. `OpenFlag` and `RenameFlag` check a name as the server does
(`nameRefusalOf`): against the flags the fake lists, and every built-in flag by its region name (`GAME_FLAG_NAMES`),
whatever its case. The tests drive the page on `FakeAdmin.flags` (`FakeFlags`), which keeps Bavaria (its capital Munich)
and Savoy (its capital Chambéry) the same way, at `AT`.
