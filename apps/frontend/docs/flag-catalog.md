# The list of flags

**The flags the game ships with are in the build. The ones opened since come from the server.**
`static/countries/countries.json` is the built-in list: the codes, their order, their pictures in the atlas and their
names in every language. The backend's flags module owns the whole catalog: `flags.v1.FlagService/ListFlags` lists the
built-in flags first, then the opened ones, each with a name in English and whether it is closed. An opened flag's code
is a flag's shape the build does not ship (`isOpenedCode`): `r-` and ten letters or digits, or the code a flag kept when
it left the build. Brittany (`xb`) shipped with the game until 2026-11-24 and is sold since, under its code, so its
land, players and room stay its own (see the backend's [flags module](../../backend/docs/flags.md#brittany-from-the-build-to-the-sold-flags)).

**A flag opened since the build is sold** (step 2 of the regional flags, 2026-11-23): the operator opens it by hand in
the backoffice, with a picture and an end. The list carries its `sold`: its picture (a PNG at most 768 x 512, between
square and 2 wide by 1 high, on `https://media.clickplanet.lol/flags/pictures/<sha>.png`, with the `stretch` and `focus`
that `flagFit.json` gives a built flag), when its time runs out, when it closed, and its pin (`Sold.pin`, where it starts
in Capital Siege, latitude and longitude, never a tile), which `SoldFlag.pin` keeps and a change of the flag starts from
(see [Flag submissions](flag-submissions.md)).

## The catalog

- **`FlagCatalog`** (`domain/flags.ts`) is the built-in flags merged with the server's list. `knows`, `playable`,
  `playableCodes` (the picker's), `openedName`, `pictureOf`, `soldOf` and `pictures` (every sold flag's, closed ones
  too). A built-in flag keeps the page's own names and its sprite, whatever the list says of it.
- **A closed flag stays known**: history and the boards still name it and draw its picture, and its land stays. But no
  player may play for it, so it is not in `playableCodes`, and the backend refuses a click or a pledge for it as for an
  unknown flag.
- **The store** (`useFlags`, `app/flags/flagsStore.ts`) holds the catalog. `useFlagsFeed`, which `PlayScreen` calls,
  serves it `Backends.flags` (`ConnectFlagsBackend`, or `FakeFlagsBackend` in fake mode) and listens to the flags
  module's stream through it (see [backends.md](backends.md#the-flags-stream)). The archive page serves it alone: it
  reads the list once, with no stream.

## When the page reads the list

- **At load**, once.
- **Each time the flags stream opens**, the first time included: nothing told before it opened is sent again.
- **Not on `flag_changed`**: the stream tells the flag as it is now, and the store merges it with no read. A flag told
  during a read is merged again over that read's answer, which may predate it.
- **When a lookup meets a code the catalog does not know**: the name of an opened code (`translator().country`), a
  `CountryFlag` with no picture, and `knowsFlag`.
- **At most one read is in flight.** A stream opening during a read queues one more read after it, never two. A code missed
  during a read waits for that read.
- **A code reads the list once.** A code still unknown after its read never starts another one. On 2026-10-31 a new
  enum value made old tabs reconnect every 500 ms, so an unknown flag must not loop like that (see
  [backends.md](backends.md)).
- **The load read may come from the browser's cache. Every later read goes past it** (a second client, with
  `cache: "no-cache"`): it reads because the list may have changed.
- **A read that fails keeps the flags the page knows**, and logs. A server with no flags module answers
  `Unimplemented`, which lists none: the built-in flags stay.

## A sold flag on the globe

**The built atlas stays as it is, 1300 x 1232, and each sold flag is drawn under it, in the same texture**
(`FlagAtlas`, `app/viewer/flagAtlas.ts`). No sold flag, no change: the texture is the built atlas `TextureLoader`
loaded.

- **Its number**: a sold flag takes a flag number of its own, after the built ones and the blank, the first time the page
  meets its code (`flagNumberOf` in `atlas.ts`, at most `SOLD_MOST`, 256; a flag past them wears the blank). The number
  never changes while the page lives, so a tile keeps it.
- **Its cell**: a built sprite's width, 100, and its picture's height at that width, 50 to 100. The cells run row by row
  from 16 rows under the built atlas: those 16 rows repeat the atlas's last row, so a sample past a bottom flag's edge
  finds its own colours, as at the texture's edge before (`soldCellOf`).
- **Its picture**: fetched with CORS, decoded to an `ImageBitmap`, drawn upside down at its cell's size in a canvas and
  made a bitmap again, all off the frame. A frame sends one picture whole (`copyTextureToTexture`), then the GPU builds
  the mip levels, and only while the camera is not moving, as the slots' cells are.
- **The texture grows** when a cell falls past its bottom: a taller texture, by steps of 256 rows so the built atlas
  keeps its texels' place in every mip level, never past 4,096, some players' limit. The built atlas (from its loaded
  image, flipped by WebGL as `TextureLoader` sent it), the gutter and every picture sent go into it in one frame. 256
  sold flags fill 3,280 rows.
- **Until its picture is in the texture** a sold flag wears the blank (`BLANK_REGION`, transparent, drawn white). Once
  it is (`showPicture`), the tiles take its cell: the compact layout's rectangle texture at once, the plain layout's tiles
  of that number one by one (`TileField.showFlag`); the landmasses it leads are framed again at its shape
  (`BorderField.showFlag`), and the slots are ranked again.
- **The far view's slots** draw a sold flag from its 768 x 512 picture where a built one is drawn from its SVG, and the
  picture's `stretch` and `focus` take the place of `flagFit.json`'s entry (see [The zoomed-out
  view](zoomed-out-view.md)).
- **A flag opened, closed or opened again while the page is open** is told by the stream and drawn with no reload. A
  closed flag stays drawn: its land stays. The globe follows the store itself (`useFlags.subscribe` in `globe.ts`), so the
  archive's and the clips' globes draw them too.

## A sold flag in the page

- **Its picture is its `picture.url`** wherever the page draws a flag as a picture: `CountryFlag` (the picker, the chat,
  the standings, the cards, the board), the share card (asked with CORS, or the card could not be saved), a rally's pin,
  and the clips (`flagPictureUrl`). Its name is the list's, in English, not translated.
- **`https://media.clickplanet.lol` must allow the page**: the bucket's pictures answer `Access-Control-Allow-Origin: *`
  (a Cloudflare rule). The game's pages send no Content-Security-Policy; one that is added must allow that origin in
  `img-src` and `connect-src`.

## A flag renamed

**A rename** comes as `flag_changed` with the new name, open or closed: the catalog takes it (`withFlags`), and the
picker and every name the translator reads from the catalog show it. Its code, picture and land stay. A name already on
screen keeps the old one until its component draws again, as a code does until the list names it.

## A held flag closed

- **`GetAllegiance` answers `closed`** (`Allegiance.closed`) when the flag held is closed and no lock of the rules holds
  it: no click plays for it, and the account changes flag at once, with no wait (`lockOf` holds nothing). While the
  rules lock the flags, as in a battle, the server keeps it playable and `closed` stays false.
- **The page reads it again** when the stream tells that the held flag closed or came back (`useHeldFlagClosings`), and
  on `rules_changed`, as it always did.
- **The window of a closed flag** (`ClosedFlagModal`) asks for another flag at once, the knock-out's out window with
  other words ("Bavaria is closed", "Pick another flag."): the flags a player may play for, a pick to confirm, and no
  close, since nothing plays for the flag held (see [The capitals](capitals.md)).

## A code the page does not know

Each place that looks up a flag by its code survives a code it does not know.

- **Its name** (`translator().country`): a built-in code reads as before (the catalogue's lines, then Intl). An opened
  code reads as the list's English name, which is not translated. Until the list names it, it reads as its code. The
  translator keeps neither in its cache. "In" an opened flag (`countryIn`) is the flag, then the name. A name already on
  screen shows the code until its component draws again.
- **Its picture** (`CountryFlag`): the atlas's sprite, a sold flag's picture, or a plain grey flag
  (`.country-flag-blank`, `--text-faint`) for a code with neither. With no code, nothing.
- **The globe**: a code with no picture yet takes a number of its own and wears the blank until one comes (above).
  Before 2026-11-23 such a tile kept the flag it had, and the globe dropped a click for such a flag.
- **The share card** draws a grey rectangle in place of a flag with no picture. **A rally's pin** keeps its plain cream
  cloth.
- **The picker and the knock-out's out window** list `playableCodes()`: no closed flag, and each opened flag by its name.
- **The leaderboard and the race** rank every code the server counts, known or not.
- **The held flag** the server tells (`GetAllegiance`) is set as it is, known or not, and so is its chat room.
- **The country kept in storage** may be a built-in code or an opened one, which the page keeps before it reads the list.
- **A share link** (`/r?f=`, see [Sharing](sharing.md)): a built-in flag is chosen before the first render, as before.
  An opened flag waits for the first read, and is chosen if the list says it is open and the player has not changed
  flag in the meantime. Any other code is ignored.
- **The standings** draw the flag of any code but an empty one. **The anthem** has no recording for it, so its play
  button is off.
- **Built-in flags alone**: a flag's old address (`clickplanet.lol/<flag>/<name>`, written into the home page at the
  build). An opened flag never had one.
- **The clips** know a sold flag when the run hands the list (`--flag-list`, see [Clips](clips.md)); without it, it is blank
  and named by its code.

## In fake mode

The fake lists Brittany sold, as the server does, open for 30 days from the page's load, its picture at its address in
the bucket (the blank until the file is there). `openFlag("Bavaria", days)` on the console opens a sold flag for that many days (30 by default), with a picture drawn
in the page from its name (three bands and a disc, 3:2, 2:1 or square, a blob URL, `paintFakeFlag`). `addTime(code,
days)` ends it later, or opens it again within 30 days of its close; `closeFlag(code, note)` closes it, with the note its founder reads; `renameFlag(code,
name)` renames it, open or closed. Each is told as the flags stream tells it, and the fake planet hears the closes
(`FakeBackend.closeFlags`): it answers `closed` for a closed flag held, unless `lockFlags()` locks the flags, and refuses
a click or a pledge for one. Nothing closes a flag when its time runs out: `closeFlag` does. `openFlag` gives a flag no
pin.

The fake keeps the server's rules on a name: `openFlag` and `renameFlag` refuse one of no character or over 40, or one
another flag has, built-in or sold, whatever its case.

A flag sent from the picker is opened by `openSubmission()`, with the upload's picture and its pin, after
`approveSubmission()`; `rejectSubmission(reason, note)` rejects it (see [Flag submissions](flag-submissions.md#in-fake-mode)).

To see the whole flow: `openFlag("Bavaria")`, pick Bavaria in the picker (its picture beside its name), take tiles and
zoom out (its picture on the tiles, then painted over the land), `closeFlag("<code>")` (the window asks for another flag,
the tiles keep the picture), then `addTime("<code>", 7)` (Bavaria is in the picker again), and `renameFlag("<code>",
"Bayern")` (the picker names it Bayern).

## Not built yet

A sold flag's name in other languages, the shop. A submission is built (step 3, see [Flag
submissions](flag-submissions.md)).
