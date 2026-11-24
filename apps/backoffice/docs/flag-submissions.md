# Flag submissions

`src/submissions/`, one page (`SubmissionsView`), in the menu under **Moderation**, after Graffiti, at
`#/submissions`. **Players submit a flag** (a name, a picture, its capital on the globe), and the operator judges it
here (step 3 of the regional flags). It is under Moderation because it is an inbox of what players send, as the reports
and the graffiti are. `flags.v1.AdminService` keeps them: `ListSubmissions`, `ApproveSubmission`, `RejectSubmission`
and `OpenSubmittedFlag`. **The capital is `pin` in the code and on the wire** (`flags.v1.Pin`); the page says capital:
it is where the flag starts in Capital Siege.

## The inbox

- **The same pattern as the reports** (see [Acting on a report](reports.md)): the list on the left and the submission
  open on the right from 1280px, one or the other below. The submission open is in the address bar
  (`#/submissions/<id>`), so a link or a reload opens it.
- **The page reads five things once**: `ListSubmissions` (every waiting one, the oldest first, then the 500 decided
  last), `ListFlags` (the catalog, to compare with), `player.v1.AdminService/GetPlayers` (the founders' names),
  `support.v1.AdminService/ListLetters` with `open_only` (the letters a player replied to, 500 at most) and
  `planet.v1.AdminService/GetTerritories` (the board the world plays, for the map). Reload reads all but the board
  again; an open reads the first three. It filters on the page: **Waiting** (the default), **To open** (approved new
  flags, which wait to be paid for), **Replies** (any submission whose founder replied to a letter) and **All**, each
  with its count.
- **A row** is the picture small, the name, "Many colours" for a likely photo, **"Replied" when its founder's reply to a
  letter waits**, how long ago, "New flag" or "Change of Bavaria", the founder, and the verdict once there is one.
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
  decided, the share of other colours, the player's capital, with a link that opens the game's globe there
  (`https://clickplanet.lol/play?lat=…&lon=…&zoom=20`, as a Discord ping does), and the flag it changed or opened.
  A rejected one shows its reason and its note.
- **The capital on a planisphere** (`CapitalMap`, see [The map](#the-map) below), with the other capitals around it.
  A waiting one's capital moves on it before the verdict (see [Moving the capital](#moving-the-capital)).
- **Capitals near it**, measured from the capital where it is now (moved or the player's), by great-circle distance,
  within 1,000 km (`NEAR_KM`): the 5 nearest (`NEAREST`) of the sold flags' capitals (`Sold.pin`) and the other new
  flags submitted, and the 5 nearest of the game's countries' capitals, each with its distance. The flag a change is
  for, a submission opened (its flag is listed) and the changes are left out. A submission is one click away.
- **The same name**: any flag, the game's own or sold, and any other submission, whose name is this one's, case and
  accents aside (`foldName`). A submission opened is left out: its flag is listed.

## The map

`CapitalMap`, over `mapView.ts` (the view: its middle, its scale, the frame) and `mapLand.ts` (the land, drawn in
pixels). **A planisphere**: longitude across, latitude up, as the Home heatmap; it goes on round the world east and
west, and stops at the poles.

- **It opens on the capital's region, 2,000 km across** at its latitude (`REGION_KM`), the country around it in the
  frame. **Zoom out to the whole world and back**: the wheel, two fingers, or + and −; **World** shows the whole world,
  **Capital** the region again. A drag pans. The frame is 2:1, 3:2 below 600 px. The caption names the board and how
  far the frame spans ("earth-2 · 2,000 km across", "the whole world").
- **The land is the board's tiles**, on the board the world plays (`GetTerritories`'s `board`): each tile an ellipse a
  little wider than the lattice's step (0.65 of it), as wide as its longitude stretches, so the coasts are the tiles'
  own. **Drawn into pixels, never one element a tile**: the whole world once a board, at 4 px a degree (1,440 × 720,
  `WORLD_BOX`), and, closer than that, the frame's own land at the screen's pixels, a quarter of the frame past each
  side, again at most every 120 ms while the view moves. A grid of the tiles by one-degree cell, made once a board,
  hands each drawing the tiles in its box alone. **On `earth-2`** (1,004,682 tiles), measured in node: the grid 7 ms,
  the whole world 60 ms once, a frame 2,000 km across at 1,400 px 3 ms, one 100° across 16 ms; in the demo the map
  shows 1.1 s after the page opens, the board's places read and rebuilt included.
- **The other capitals are marks over it**: the game's countries' (grey, from the board's capitals file, see
  [Boards on the page](boards.md)), the sold flags' (blue) and the other new flags submitted (a blue ring); the
  submission's own is red, with its name always. **Names show with their flag** (the page's `FlagMark`, a submission's
  picture for one not yet a flag) on a frame narrower than 90°, never over one another nor under the buttons: the
  flags' and the submissions' first, then the countries' nearest the middle; beside the mark, or before it at the
  frame's right edge. On the whole world, the marks alone.

## Moving the capital

On a waiting submission alone; a decided one's map only reads.

- **Drag the red mark**, or **tap the map** where the capital goes. A tap is a press that moves less than 6 px; a drag
  that moves more pans. The place is rounded to four decimals (about ten metres): what the page shows is what it sends.
- Under the map: "Moved to 42.3619, 5.0922 · Open on the globe" and **Back to the player's**. Capitals near it are
  measured from there.
- **Approve sends it** (`ApproveSubmissionRequest.pin`; unset keeps the player's). A change wears it on its flag too.

## The verdict

`SubmissionVerdict`, on a waiting one alone. Each action asks first (see [Every action](actions.md)).

- **Approve**: "Approve Catalonia? Its picture becomes public, and the flag waits to be opened." With the capital
  moved: "Approve Catalonia, its capital moved to 42.3619, 5.0922? …". For a change: "Approve the new name, picture and
  capital of Bavaria? Every player sees them at once." (moved: "Approve the new name and picture of Bavaria, its
  capital moved to …?").
- **Reject**: a reason from the list (Not a place's flag, Has a flag already, Unreadable on a tile, Hateful, Other)
  and a note of 500 characters at most. Other needs a note. The page says plainly that the player reads the reason and
  the note, and the question says it again.
- **What the server refuses**: `not_found` is a submission no longer kept; **`failed_precondition` is quoted** ("The
  server refused it: the flag r-… is named Tyrol"): one decided already, a name another flag took since, the flags'
  pictures off; `invalid_argument` is quoted too (a capital off the globe).

## Writing to the player

`LettersPanel` (in `src/support/`, over `letters.ts`), "Write to SilentEagle29", on every submission whose founder is
an account. **A letter is a mail the operator writes first**, from support@clickplanet.lol, to the account's verified
address; the player's replies come back to it (`support.v1.AdminService`: `WriteLetter`, `ListLetters`, `CloseLetter`;
the backend's [Writing to a player first](../../backend/docs/support.md#writing-to-a-player-first)).

- **The founder's letters** (`ListLetters` with its account id), the latest written or replied to first, each a thread:
  its subject, its address, then its mails ("ClickPlanet"), the player's replies and its last close, by time, as a
  note's conversation (see [Answering a note](notes.md)). A reply waiting for the operator is marked **Replied**.
- **Write again**, in a letter: one more mail in its thread. **Close**, on a letter with a reply that waits: it waits
  for nothing until the player writes again.
- **New letter**: a subject, "Your flag Catalonia" at first (1 to 200 characters), and the text. **Send asks first and
  names the address** when a letter to the player gave it ("Send this to ada@example.com?"), or else "the player's
  verified address"; then it says where it went ("Sent to ada@example.com.").
- **How the operator sees a reply**: the row's **Replied** mark, and the **Replies** filter, from `ListLetters`'s
  `open_only` (a reply after the last mail, no close after it), read with the list and again after each send or close.
- **What the server refuses**: `failed_precondition` is no verified address, or a letter past 90 days; `unavailable`
  is a mail that did not go, and nothing was kept: send again; `invalid_argument` is quoted; `not_found` a letter no
  longer kept. The text typed stays.

## Opening an approved new flag

`OpenSubmittedPanel`, on an approved new flag alone (state Approved, not a change). The flag waits to be paid for. Then
the operator opens it: an end, a `datetime-local` in the operator's own time zone, 30 days from now at first, as the
Flags page's form. **Open asks first**: every player sees the flag at once and may play for it, and a flag is never
deleted. `OpenSubmittedFlag` opens it at the submission's capital. The green line says its code and its end, and the page
reads the lists again: the submission is then Opened, with its flag.

## Hosts

A waiting or rejected submission's picture is a signed link to the private bucket, on the R2 host already in `img-src`;
an approved one's is on `https://media.clickplanet.lol`, also there (see [Deploy](deploy.md)). Nothing to change in
`public/_headers`. The globe link opens a new tab, which the CSP does not limit.

## In the demo and the tests

`npm run demo` holds seven made-up submissions, each picture drawn in the page (SVG): Catalonia (waiting, fine), Tyrol
(waiting, a name the closed sold flag Tyrol has, its capital beside it, and a letter its founder replied to), Lake Garda
(waiting, a photo, 41% of other colours, a founder with no verified address), Free State of Bavaria (waiting, a change
of Bavaria), Sardinia (approved, to open), Brittany (rejected: Has a flag already) and Cornwall (opened). The map is
drawn on `earth-2`, the board the world plays. `FakeAdmin.flags` (`FakeFlags`) keeps six of the same kind for the
tests, and answers the four calls with the server's refusals: `not_found` for an unknown id, `invalid_argument` for a
capital off the globe, `failed_precondition` for one decided already, a name another flag has, or not an approved new
flag, `invalid_argument` for no reason, Other with no note, a note past 500 characters, or an end gone by. An approval
keeps the moved capital on the submission, and on its flag for a change. `FakeAdmin` answers the letters as the server
does: a verified address for BraveFox42 alone (`addresses`), `failed_precondition` for none, `invalid_argument` for a
subject or a text out of bounds, `not_found` for an unknown letter. The tests draw the map on a patch of `earth-1`
(jsdom draws no canvas: they read the marks, the names and the caption).
