# Season standings

`backends/standings.ts` is the contract: `StandingsBackend`, `Standing` (a
ranked player: rank, its account, the flag its tiles are for, tiles; its name and color are the nameplates', see
[Nameplates](nameplates.md))
and `MySeason` (the caller's line on one board: that flag, its tiles, its rank),
scoring's `Round` (live standings while open, results once
scored), and the countries' `Race` (the book's `Score`s, today's round and the
running battle's). Scoring knows no season, day or battle: every read takes the
gamemaster's ids, the season's `bookId` and the rounds' `roundId`.
`standingsBackend.ts` implements it over `scoring.v1.ScoringService`
(`ListenForEvents`, `GetMyTally`, `GetRound`, `GetScores`), and
`fakeStandingsBackend.ts` stands in for it in fake mode, counting the player's
own clicks and moving the other players a few tiles every 1.5s. `Boards`
(`boardsOf`) is the players' boards of one book, which `PlayScreen` makes for the
current season's book (the ended season's, from its debrief, until another runs).
With no book there are no boards. `app/standings/` draws it.

- **A player's season is the tiles it took this season for its main flag**, the
  flag it took the most for: that is the Players board. **A country's board is
  every player who took tiles for that country, by those tiles**, so a player is
  on the board of each flag it took for, from its first take. The server ranks
  only signed-in players (each has a username; a guest has none), and ties share
  a rank (1, 2, 2, 4). `RankCoin` draws the rank, as on the countries' board.
- **The board has three views** (`BoardViews`): Countries, the `Leaderboard`;
  Players; and the players of the country played for, named by its flag
  and name. `PlayScreen` holds the view, so a closed sheet or
  another menu tab keeps it. With no `StandingsBackend` wired the board has no
  views.
- **The countries' points are on the Countries board, where everybody looks.**
  Each day is a round, the top 10 countries by the ground they held score 25, 18,
  15 … 1, and a battle's round its times that (the Final Battle ×3; the
  backend's [Scoring](../../backend/docs/scoring.md)).
  `listenForRace` follows the stream for the book and the rounds asked for, today's
  (`Timeline.today.round_id`) and the running battle's (`Turn.round_id`), and reads
  its `scores` and `round` cases. **The race is a store**, `useRace`
  (`app/standings/raceStore.ts`, see [Stores](stores.md)): it keeps the book's points
  through new round ids and a round only while the same ones are asked, and only while
  it is open (`raceOf`). `useRaceFeed`, which `PlayScreen` calls once with the backend and
  the ids, follows the stream into it, again when either id changes, and leaves it
  empty once it stops. `Leaderboard`, `Menu` and `BoardSheet` read it themselves;
  `PlayScreen` reads it only for the order it ranks by, which it hands the anthem, the
  status bar's rank and the shared picture. With a race the
  board adds two columns: Points, each country's points from
  the days closed, right-aligned, and Today, what it scores if the day ends now, in
  green ("+18") and left-aligned. **While a round runs, "% of map" becomes Gained**:
  the tiles the country holds now less what it held when the round opened (`Round.start`,
  `countryLines`' `gain`), signed, green up and red down (`--gain`, `--loss`), with its
  own hint (`gained`, `gainedBattle`). A round ranks that gain (see the backend's
  [Scoring](../../backend/docs/scoring.md#the-rules)), so the number beside the points
  is the one that earns them; it moves with the tiles, live, while the points follow
  the round's read every 10s. The owner asked for it on 2026-11-06: the points alone
  did not say why a smaller flag won the day. **Today is its own column so the points line up**:
  as a badge beside the points it pushed only the rows that had one. **While a
  battle's round is open** (`race.battle`, `scoringNow`), the column is
  the battle's: crossed swords and its times ("×3", in `--war`) for "Today", its own
  hint (`battle`), and its points, already ×times, read "this battle" to a screen
  reader. The leader
  frame shows the points where it showed the share, and its gain ("+120 today", "-10
  this battle") beside its tiles, first, or its share while no round runs, so the tiles' delta badge floats into the room after them. On a narrow
  phone the two wrap onto two lines rather than run into the points.
- **Each number says what it means** (`Hint` in `Leaderboard`, the texts in
  `boardFigures.ts`), as a bonus does: the column heads and the leader frame's
  figures are buttons that show a `Bubble`: with a mouse, for as long as it is
  over them, a click included; after a tap or a key, for `TAP_HINT_MS` (7s), long
  enough to read, or until the next touch anywhere, its own head included, which puts it
  away (a `pointerdown` on the document; a mouse's is ignored, since its bubble follows the
  mouse). Before 2026-11-07 a phone had no way to close it but to wait. **The first time the board shows points, it says what they are
  on its own** (`GUIDE_MS`, 10s), once per browser: the store `useBoardGuide`
  (`boardGuideStore.ts`), which the board reads, in the `clickplanet-board-guide` key. The bubble renders beside its button, not inside:
  inside, it mounts before the button's ref is set and has nothing to sit on.
- **The desktop menu is `clamp(380px, 34vw, 460px)` wide**, every tab alike, for the
  six columns. It stops at 380px below about 1120px so it stays clear of the season
  chip at the top centre. On a phone the sheet is the screen's width; a long
  country name ends in "…" before a column moves.
- **The column heads sort the countries**: Tiles, Gained and Points (`Leaderboard`'s
  `onOrder`, `aria-sort` on the head sorted by). Each has a small arrow before its word
  (`.leaderboard-sort`), faint, and in `--text` on the head sorted by; before the word
  so the word stays over its column's right-aligned numbers. Points first. A tap on a
  head sorts by it and, as on any head, says what it means. Country and Today do not
  sort: the board ranks by a number, not a name, and Today follows Gained, since a
  round scores the gain. "% of map" does not either: it is the order of Tiles. It
  replaced a switch of two pills, Season and Territory, on the heading's line, on
  2026-11-06: once a round scored the gain, a third order was one pill too many.
  Points ranks by the points of the days closed, then by tiles (`domain/race.ts`,
  `countryLines`), and lists a country with points that holds no ground; Gained ranks
  the same countries by the tiles gained since the round opened, then by tiles. **Today's
  points do not move the Points order**: when a day was won on its average, ranked
  by them, two countries at 0 points swapped against their tiles, which read as a bug
  (2026-10-08). Tiles is the map's order. The choice is kept per browser: the store
  `useCountryOrder` (`countryOrderStore.ts`), in the `clickplanet-country-order` key, which
  `PlayScreen` reads; before 2026-11-07 a reload put it back to Points. With no round
  open, Gained falls back to Points; with no race, no head sorts and the board is
  the order of Tiles (`shownOrder`). **The rank and the anthem follow the order picked**: the rank
  beside "playing for" and on the phone's status bar is the country's place in
  that order (`rankShown` in `Menu`, `rankOf` over `countryLines` in `PlayScreen`),
  the shared picture's rank too (see [Sharing the globe](sharing.md)),
  and the music is its first country's (see [The leader's
  anthem](sound.md#the-leaders-anthem)). On the map's order alone, the Season board put
  France 2nd under "France #4", and played Portugal's anthem in Poland's frame
  (2026-10-12).
- **On a phone the heading is the sheet's head** (`BoardSheet`, `BoardHeading`):
  the view picked takes the place of the "Leaderboard" title, which
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
  view shown (`book_id`, and `country_id`, empty for every player) through `openStream`, with
  no token and `NO_TIMEOUT`, and stops when the view changes or closes. The
  server sends the view's whole top 10 when the stream opens and again each time
  it changes, at most once a second; a `board` replaces what is shown, a
  `heartbeat` is skipped. While the stream is down the last board stays, and a
  reconnect starts with a whole board. A server without the stream is retried
  with `openStream`'s backoff. `GetBoard` is not called.
- **`GetMyTally` reads as the identity token** (`identity()`: a fresh token, or
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
  with no render of `PlayScreen` per click. `useOwnTakes` counts them by flag, and
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
  A row of another account is another player's, whatever its name. With no
  account id read yet there is no own row. The other rows move with the
  stream: a `TileUpdate` does not say who took the tile.
- **The caller's own line.** "Your season" sits over the table and shows only
  the board on screen: on Players the season's tiles and, with a username, the
  rank among all players; on a country's board the tiles taken for that country
  and the rank there, under its name. No tile names another country than the
  one shown. A guest gets a Sign in button beside its tiles, which opens
  `SignInPitchModal`. In the table the caller's row is the same line: marked
  when it is in the top 10, and otherwise added under it with its rank there.
  Its name and color come from the nameplates store, which takes the profile's and each pick at once, and its title
  from the badges store, both by `caller.accountId`, as every row's by its `accountId` (see [Badges](badges.md)): a
  name or a title picked shows at once, a title earned once its badge is read again.
- **A name opens the player card**, as in the roster and the chat, with the
  flag of its row.
- **The head's line is a background**, 7px above its bottom edge: the gap under
  it was the first row's top padding, and the caller's highlight took it in
  when it was first.
