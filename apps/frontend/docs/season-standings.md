# Season standings

`backends/standings.ts` is the contract: `StandingsBackend`, `Standing` (a
ranked player: rank, name, color, worn title, the flag its tiles are for, tiles)
and `MySeason` (the caller's line on one board: that flag, its tiles, its rank
and its worn title), and the countries' `Race` (the `Round` in progress and the
season's `Score`s). `standingsBackend.ts` implements it over
`seasons.v1.SeasonService/ListenForEvents` and `GetMySeason`, and
`fakeStandingsBackend.ts` stands in for it in fake mode, counting the player's
own clicks and moving the other players a few tiles every 1.5s. `app/standings/`
draws it.

- **A player's season is the tiles it took this season for its main flag**, the
  flag it took the most for: that is the Players board. **A country's board is
  every player who took tiles for that country, by those tiles**, so a player is
  on the board of each flag it took for, from its first take. The server ranks
  only signed-in players (each has a username; a guest has none), and ties share
  a rank (1, 2, 2, 4). `RankCoin` draws the rank, as on the countries' board.
- **The board has three views** (`BoardViews`): Countries, the `Leaderboard`;
  Players; and the players of the country played for, named by its flag
  and name. `Viewer` holds the view, so a closed sheet or
  another menu tab keeps it. With no `StandingsBackend` wired the board has no
  views.
- **The countries' points are on the Countries board, where everybody looks.**
  Each day is a round, the top 10 countries by the ground they held score 25, 18,
  15 … 1, and the Final Battle triple (the backend's [Seasons](../../backend/docs/seasons.md)).
  `listenForRace` follows the whole map's season stream and reads its `race`
  case, which every stream carries; the server reads it again every 10s.
  `Viewer` follows it once (`useRace`) and hands it to the board. With a race the
  board keeps "% of map" and adds two columns: Points, each country's points from
  the days closed, right-aligned, and Today, what it scores if the day ends now, in
  green ("+18") and left-aligned. **Today is its own column so the points line up**:
  as a badge beside the points it pushed only the rows that had one. The leader
  frame shows the points where it showed the share, and the share beside its tiles,
  first, so the tiles' delta badge floats into the room after them. On a narrow
  phone the two wrap onto two lines rather than run into the points.
- **Each number says what it means** (`Hint` in `Leaderboard`, the texts in
  `boardFigures.ts`), as a bonus does: the column heads and the leader frame's
  figures are buttons that show a `Bubble`: with a mouse, for as long as it is
  over them, a click included; after a tap or a key, for `TAP_HINT_MS` (7s), long
  enough to read. **The first time the board shows points, it says what they are
  on its own** (`GUIDE_MS`, 10s), once per browser: `useBoardGuide`, the
  `clickplanet-board-guide` key. The bubble renders beside its button, not inside:
  inside, it mounts before the button's ref is set and has nothing to sit on.
- **The desktop menu is `clamp(380px, 34vw, 460px)` wide**, every tab alike, for the
  six columns. It stops at 380px below about 1120px so it stays clear of the season
  chip at the top centre. On a phone the sheet is the screen's width; a long
  country name ends in "…" before a column moves.
- **A switch orders the countries, Season or Territory** (`OrderSwitch`,
  `.leaderboard-order`, `aria-pressed`), Season first. It is a small pill on the
  heading's line, beside the view picked, and only on the Countries view: a row of
  its own with two full buttons cost the table two of its rows on a phone. Season ranks by the points of the days closed, then
  by tiles (`domain/race.ts`, `countryLines`), and lists a country with points that
  holds no ground. **Today's points do not move the order**: they are what the
  tiles held now would score, so they follow the tiles, which already break the
  ties. When a day was won on its average, ranked by them, two countries at 0
  points swapped against their tiles, which read as a bug (2026-10-08).
  Territory is the order by tiles held now, as the board was. `Viewer` holds the
  choice. With no race, there is no switch and the board is as it was
  (`shownOrder`). **The rank and the anthem follow the order picked**: the rank
  beside "playing for" and on the phone's status bar is the country's place in
  that order (`rankShown` in `Menu`, `rankOf` over `countryLines` in `Viewer`),
  the shared picture's rank too (see [Sharing the globe](sharing.md)),
  and the music is its first country's (see [The leader's
  anthem](sound.md#the-leaders-anthem)). On the map's order alone, the Season board put
  France 2nd under "France #4", and played Portugal's anthem in Poland's frame
  (2026-10-12).
- **On a phone the heading is the sheet's head** (`BoardSheet`, `BoardHeading`):
  the view picked and the order take the place of the "Leaderboard" title, which
  stays for a screen reader, and `BoardViews` is `headless`. Under it "playing
  for" is one line with Change and no box, since the status bar right above
  already names the country and its rank. The leader's frame is as tight as it
  goes and says nothing of the toll: the dock shows the slowdown of the country
  played for, and `ClicksPanel` the whole table. All of it so a phone with its
  browser's toolbars (about 390 × 664) shows eight rows of the table, not one
  (2026-10-09). **The whole sheet scrolls there, not the table alone**
  (`.board-sheet` in `Menu.css`): the table's column heads stick to the top, and
  "playing for" and the leader's frame go up out of the way, so a scrolled board
  shows about seventeen rows. The fade at the foot sticks to the sheet's bottom
  edge. The desktop menu still scrolls the table alone, under the leader.
- **The view is picked from the board's heading** (`HeadingSelect`, a gold
  section title that opens a listbox), not from tabs: the board is already a
  tab of the menu, and tabs in a tab read as one row of places. **The list is
  `position: fixed` under its button**: absolute, it was clipped by the menu's
  scrolling body and made a short board scroll. It follows the button every
  frame (a sheet grows upward as the board loads), opens upward with no room
  below, and closes when something that holds it scrolls.
- **The board is streamed.** `useStandings` follows `ListenForEvents` for the
  view shown (`country_id`, empty for every player) through `openStream`, with
  no token and `NO_TIMEOUT`, and stops when the view changes or closes. The
  server sends the view's whole top 10 when the stream opens and again each time
  it changes, at most once a second; a `board` replaces what is shown, a
  `heartbeat` is skipped. While the stream is down the last board stays, and a
  reconnect starts with a whole board. A server without the stream is retried
  with `openStream`'s backoff. `GetStandings` is no longer called.
- **`GetMySeason` reads as the identity token** (`identity()`: a fresh token, or
  one resumed from the cookie, never a Turnstile mint), so a player back the next
  day sees its season at once. With none to be had it is not sent, and
  `unauthenticated` reads as unknown and keeps the token. **It is asked for the
  board on screen**: no country on Players, which reads the main flag, its tiles
  and the global rank; the country on its board, which reads the tiles taken for
  it and the rank there. `useMySeason` reads it when a players' view opens, when
  the view's country changes (and shows nothing of the other board meanwhile),
  when the account or the username changes, 2s after the last of a run of
  accepted clicks, and at least every 10s while they keep coming
  (`useReadsAfterClicks`).
- **The caller's own numbers move on every take.** The globe already knows
  which click takes a tile (one on a tile its flag does not hold and no shield
  stands on), and once
  the server accepts one it tells `acceptedClicks`, which tells its listeners
  with no render of `Viewer` per click. `useOwnTakes` counts them by flag, and
  `liveSeason` adds the ones made since the read was sent to the line's flag:
  the country shown on its board, the main flag on Players. A player with no
  main flag yet gets the server's rule there (the first flag, until another has
  strictly more). On Players a take for another flag adds nothing: only the
  server knows whether it became the main one. A spread or an enclosure's extra
  tiles, and the ranks, wait for the next read.
- **The caller's row is drawn from that count** (`boardWith`): it climbs into
  the top 10 and pushes the last out, its rank counted among the rows it passes,
  and the rank in "Your season" follows it. **The caller's row is the one with
  its `accountId`** (`Caller.accountId`, from the `AccountStore`): `boardWith`
  replaces that row, `PlayerStandings` marks it, and the rows are keyed by it.
  A row that only shares the caller's name is another player's. With no
  account id read yet there is no own row. The other rows move with the
  stream: a `TileUpdate` does not say who took the tile.
- **The caller's own line.** "Your season" sits over the table and shows only
  the board on screen: on Players the season's tiles and, with a username, the
  rank among all players; on a country's board the tiles taken for that country
  and the rank there, under its name. No tile names another country than the
  one shown. A guest gets a Sign in button beside its tiles, which opens
  `SignInPitchModal`. In the table the caller's row is the same line: marked
  when it is in the top 10, and otherwise added under it with its rank there.
  Its name and color come from the profile, its title from `GetMySeason`: read
  again after clicks, it follows a rank-up, and a title picked meanwhile shows
  at the next read.
- **A name opens the player card**, as in the roster and the chat, with the
  flag of its row and its title.
- **The head's line is a background**, 7px above its bottom edge: the gap under
  it was the first row's top padding, and the caller's highlight took it in
  when it was first.
