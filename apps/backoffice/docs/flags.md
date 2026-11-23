# Flags: the sold flags

`src/flags/`, one page (`FlagsPage`), in the menu under Ops, at `#/flags`. **A sold flag is a real region's flag the
game did not ship with**, opened for every player until its end (step 2 of the regional flags: the owner opens one by
hand, for free). `flags.v1.AdminService` keeps them (`ListFlags`, `CreateFlagUpload`, `OpenFlag`, `AddTime`,
`CloseFlag`; the backend's flags module). The game's own flags are not on this page: they never end.

## Opening one

- **The form** (`OpenPanel`): a name of 1 to 40 characters, counted as it is typed; the end, a `datetime-local` in the
  operator's own time zone, which the page names, 30 days from now at first, with the days left beside it; and one
  picture (`image/png`, `image/jpeg` or `image/webp`, 5 MB at most), shown as picked. What the server would refuse is
  in red as it is typed (a name too long, another type, a file too big, an end gone by), and **Open** stays disabled
  until the three are there and none is refused.
- **Open asks first**, and the question says what it costs: every player sees the flag at once and may play for it,
  and a flag is never deleted: closed, it stays in the list for good. Only Yes uploads anything.
- **Then three steps** (`openFlag` in `actions.ts`): `CreateFlagUpload` with the file's type and size, which answers an
  upload id and a signed link; one `PUT` of the file to that link with the same `Content-Type` (`putFile`, as a
  video's, see [Videos](videos.md)), with a progress bar; then `OpenFlag` with the name, the upload id and the end. The
  server draws the picture again and makes it public. The page says "Opened Rennes (r-…), until …", empties the form
  and reads the list again.
- **A refused step says why**: `CreateFlagUpload`'s and `OpenFlag`'s `failed_precondition` is "The flags' pictures are
  off on the server"; `invalid_argument` (a picture that moves, narrower than square or wider than 2 by 1, a name, an
  end) quotes the server; a failed `PUT` quotes the bucket. A try again sends the picture again: an upload link is
  cheap.

## The list

- **Every sold flag** (`ListFlags`, the game's own left out), each a card: its picture as the server drew it, its
  name, its code with Copy, and Open or Closed. **An open flag** shows its end and the days left (to the nearest day,
  hours in its last two). **A closed one** shows when it closed, and until when time added opens it again (30 days
  after the close), or that it no longer can.
- **By end** (open first, the soonest to end first, then the closed ones, the last closed first) or **By name**. The
  contract has no time a flag opened, so there is no newest first.
- **Each read also names the flags to the rest of the console** (`learnFlags`, see
  [A country on the page](countries.md)): a flag opened here shows its picture and name on the other pages at once.
- **The list as last read stays while it is read again**, so each card keeps what the server answered it.

## Days and closing

- **Add days** (a whole number from 1 to 3660, 30 at first) asks first. For an open flag: "Add 10 days to Bavaria? It
  ends on … instead of …". For one closed less than 30 days ago: "Open Savoy again for 30 days, until …?", and every
  player sees it again with the land it still holds. Then it says the new end ("Open again. It ends on …") and reads
  the list again. **Past the 30 days it is disabled**; the server's `failed_precondition` says the same.
- **Close**, on an open flag, asks first: its players must pick another flag, nobody may click for it, its land stays;
  in a battle that locks the flags it plays on until the lock lifts; time added within 30 days opens it again. Then it
  says until when, and reads the list again. Its `failed_precondition` is "It is closed already".

## Hosts

The upload goes to the R2 endpoint already in `connect-src`, and the pictures load from `https://media.clickplanet.lol`,
already in `img-src` (see [Deploy](deploy.md)): nothing to change in `public/_headers`.

## In the demo

`npm run demo`: Bavaria and Cornwall open (Cornwall in its last day), Savoy closed 3 days ago, Tyrol closed 41 days
ago, each picture drawn in the page (SVG), and a few made-up players for Bavaria, so the other pages show its picture.
The made-up bucket takes a picture in two seconds; `OpenFlag` refuses a picture it does not hold, a name of more than
40 characters and an end gone by, and the flag shows its picture from the page's memory until a reload. The tests
drive the page on `FakeAdmin.flags` (`FakeFlags`), which keeps Bavaria and Savoy the same way, at `AT`.
