# Flag submissions

`src/submissions/`, one page (`SubmissionsView`), in the menu under **Moderation**, after Graffiti, at
`#/submissions`. **Players submit a flag** (a name, a picture, a pin on the globe), and the operator judges it here
(step 3 of the regional flags). It is under Moderation because it is an inbox of what players send, as the reports and
the graffiti are. `flags.v1.AdminService` keeps them: `ListSubmissions`, `ApproveSubmission`, `RejectSubmission` and
`OpenSubmittedFlag`.

## The inbox

- **The same pattern as the reports** (see [Acting on a report](reports.md)): the list on the left and the submission
  open on the right from 1280px, one or the other below. The submission open is in the address bar
  (`#/submissions/<id>`), so a link or a reload opens it.
- **The page reads four things once**: `ListSubmissions` (every waiting one, the oldest first, then the 500 decided
  last), `ListFlags` (the catalog, to compare with), `player.v1.AdminService/GetPlayers` (the founders' names) and
  `planet.v1.AdminService/GetTerritories` (the board the world plays, for the map). Reload and an open read the first
  three again. It filters on the page: **Waiting** (the default), **To open** (approved new flags, which wait to be paid for) and
  **All**, each with its count.
- **A row** is the picture small, the name, "Many colours" for a likely photo, how long ago, "New flag" or "Change of
  Bavaria", the founder, and the verdict once there is one.
- **After a verdict the next waiting one opens** (the one below, or the first one left), and a green line says what
  was done. A submission picked by hand clears the line.

## The submission open

Top down, as it is judged (`SubmissionDetail`):

- **A warning for a likely photo**: `other_colors_percent` is the share of the picture in colours past its 8 most
  common. From `PHOTO_FROM_PERCENT` (15, in `submissions.ts`) the page says it is most likely a photo. The game's own
  flags give 0 to 10, photos 24 to 65.
- **The picture as the server drew it**, at its own size (up to 768 × 512), then **as a tile shows it**
  (`FlagFigure`): a disc with the middle of the picture, its height whole, 50 px and 16 px wide (`TILE_SIZES`). This
  is the game page's own drawing: its display shader crops the flag to a square on its middle and cuts a disc from it.
  A tile is about 50 CSS px wide at the closest zoom on a phone, 75 on a screen 1,000 px high. Before the approval the
  picture is a private link, good for an hour: a link that ran out says to reload the list. A rejected picture is
  deleted after 30 days: the page says so.
- **For a change**: the flag it changes, with its picture and name now ("Now: Bavaria"), beside the new ones ("Asked:
  …").
- **The name, the founder and the facts**: the founder's name opens the player page, and its account id has Copy. An
  account the player read does not name (a deleted one keeps its id) shows its id alone. Then when it was submitted and
  decided, the share of other colours, the pin, with a link that opens the game's globe there
  (`https://clickplanet.lol/play?lat=…&lon=…&zoom=20`, as a Discord ping does), and the flag it changed or opened.
  A rejected one shows its reason and its note.
- **The pin on a map** (`PinMap`): the land around the pin, 500 km across, north up, drawn as the graffiti are (see
  [Graffiti](graffiti.md), `groundOf` in `graffiti/picture.ts`), on the board the world plays
  (`planet.v1.AdminService/GetTerritories`, its `board`). The pin is red, and the other pins in the frame are blue.
- **Near it**: the sold flags' pins (`Sold.pin`) and the other new flags submitted, by great-circle distance, the 5
  nearest within 1,000 km (`NEAREST`, `NEAR_KM`). The flag a change is for, a submission opened (its flag is listed)
  and the changes are left out. A submission is one click away.
- **The same name**: any flag, the game's own or sold, and any other submission, whose name is this one's, case and
  accents aside (`foldName`). A submission opened is left out: its flag is listed.

## The verdict

`SubmissionVerdict`, on a waiting one alone. Each action asks first (see [Every action](actions.md)).

- **Approve**: "Approve Catalonia? Its picture becomes public, and the flag waits to be opened." For a change: "Approve
  the new name and picture of Bavaria? Every player sees them at once."
- **Reject**: a reason from the list (Not a place's flag, Has a flag already, Unreadable on a tile, Hateful, Other)
  and a note of 500 characters at most. Other needs a note. The page says plainly that the player reads the reason and
  the note, and the question says it again.
- **What the server refuses**: `not_found` is a submission no longer kept; `failed_precondition` is one decided
  already, or (for an approval) the flags' pictures off on the server; `invalid_argument` is quoted.

## Opening an approved new flag

`OpenSubmittedPanel`, on an approved new flag alone (state Approved, not a change). The flag waits to be paid for. Then
the operator opens it: an end, a `datetime-local` in the operator's own time zone, 30 days from now at first, as the
Flags page's form. **Open asks first**: every player sees the flag at once and may play for it, and a flag is never
deleted. `OpenSubmittedFlag` opens it at the submission's pin. The green line says its code and its end, and the page
reads the lists again: the submission is then Opened, with its flag.

## Hosts

A waiting or rejected submission's picture is a signed link to the private bucket, on the R2 host already in `img-src`;
an approved one's is on `https://media.clickplanet.lol`, also there (see [Deploy](deploy.md)). Nothing to change in
`public/_headers`. The globe link opens a new tab, which the CSP does not limit.

## In the demo and the tests

`npm run demo` holds seven made-up submissions, each picture drawn in the page (SVG): Catalonia (waiting, fine), Tyrol
(waiting, a name the closed sold flag Tyrol has, pinned beside it), Lake Garda (waiting, a photo, 41% of other
colours), Free State of Bavaria (waiting, a change of Bavaria), Sardinia (approved, to open), Brittany (rejected: Has a
flag already) and Cornwall (opened). `FakeAdmin.flags` (`FakeFlags`) keeps six of the same kind for the tests, and
answers the four calls with the server's refusals: `not_found` for an unknown id, `failed_precondition` for one decided
already or not an approved new flag, `invalid_argument` for no reason, Other with no note, a note past 500 characters,
or an end gone by.
