# The board the page draws, and the reload when it changes

**The game page draws the board `GetMap` names as it loads, and reloads when the server plays another** (the owner,
2026-11-17: a reload, not a rebuild in place). A board is a geometry, `earth-1` today (see
[Boards](../../../docs/boards.md)); every tile id the page reads, clicks or hears is a tile of the board it drew, so a page
on the wrong board paints every take on other ground. The reload is the same move as a server restart's
([A server restart reloads the page](../../../docs/reload-on-restart.md)): the new page loads the board, and the bundle
that carries it.

## At the boot

- **`BoardChanges.choose`** (`backends/boards.ts`) reads the board from `PlanetBackend.readBoard`: a `GetMap` of the
  first tile alone, which every board has, so the read is never off the map for the antibot's `scraper`. `main.tsx`
  starts it at once, beside the language and CrazyGames, and hands the page `Backends.board`, a promise `useGlobe` gives
  `createGlobe`. The globe loads that board's runs, borders, border lines and terrain, and the page that board's
  landmass names (`landmassNamesOf`, [Translation](i18n.md)).
- **An answer that names no board** is from a server before boards were named: `earth-1`.
- **The stream's opening is trusted over the read.** `PlanetBackend` opens the planet stream as it is built, and its
  first heartbeat names the board. A stream is never cached; a heartbeat heard before the choice is the board drawn.

## Later answers

- **The stream opens on another board** (`BoardChanges.opened`, from the first heartbeat of each connection): the page
  reloads at once, and the opening is not told to the globe, so it reads nothing of the new board. A change of board is
  a new epoch that cuts every stream, and the stream opens again 500 ms later, so the page reloads within a second of
  the change: inside an arena's 8-second freeze.
- **A `GetMap` batch names another board** (`BoardChanges.answered`): it is not handed to the globe, which reads the
  range again when it finds it unread (`READ_AGAIN_MS`, 5 s, the cache's max-age). If that read, made past the browser's
  copy, still names another board, the page reloads. A batch of the drawn board in between is the browser's copy it was,
  and the wait starts over.

## No loop

**`GetMap` is a GET the browser keeps 5 s** (`Cache-Control: max-age=5`), so a page that reloads for a new board can read
the old one from the cache. A reload leaves its board and its time in the tab's `sessionStorage`
(`clickplanet-board-reload`):

- **After a reload for a board**, a boot that reads another reads `GetMap` again every second, for 6 s at most, until it
  names the one the reload was for. Past that it takes what the server says: the server moved on, and the page does not
  reload again.
- **A board this bundle does not carry** is a stale bundle: the page reloads once for a newer build. If the reloaded page
  reads the same board, `choose` throws `StaleBundleError` and the globe shows its load failure instead of reloading
  again.
- **The memory lasts a minute**: a page loaded later is a page of its own, and may reload once again. It is forgotten
  once a board is drawn. A browser with no storage reloads at most once per load, and reads `GetMap` as a first load does.
- **`reload` is called once** per page, whatever is heard after.

## The world grew

**A browser that saw the world before its switch of board sees "The world grew" once**, on the first page it draws on
the new board: a full-screen moment in the announcements queue (`WorldGrew`, `{kind: "worldGrew"}`, see
[Battles](battle.md#the-announcements-queue)), the title reveal's stage with the globe, "4× more ground" (the new
board's tiles over `earth-1`'s, rounded) and "Every country kept its land.", and Close.

- **The rule is `domain/worldGrew.ts`**: the moment plays on `WORLD_GROWN` (`earth-2`) alone, never on an arena's board,
  never on a first visit (`guided`: that browser never saw the world smaller, and the board is kept as seen at once),
  and never twice: `useWorldGrew` keeps the board in local storage (`clickplanet-world-grew`) when the player closes
  it. A browser with no storage sees it on each page.
- **The page needs no word from the server**: the switch cuts every stream, and the page reloads onto the board the
  stream names (above). The World chat says it too (`board_switched`, see [Live chat](chat.md)), as does Discord.
- **To see it in fake mode**: `/play?fakeBoard=earth-2` on a browser that has visited before, after
  `localStorage.removeItem("clickplanet-world-grew")`.

## In fake mode

`VITE_FAKE_BACKEND=1`: `FakeBackend.readBoard` names `earth-1`, its stream's opening names it too, and its batches pass
through the same `BoardChanges`, so the fake never reloads.

## Where it is tested

`backends/boards.test.ts` (the boot, the cache after a reload, the unknown board once, the stream's opening, a batch
read again), and the `PlanetBackend's board` cases of `planetBackend.test.ts`.
