# Who holds a tile

**A mouse resting on a tile for half a second, or a finger holding one still for
half a second, opens a card beside the pointer on the player whose take holds
it** (`HolderCard`). One row: the medal of the title it wears (its name only as
the medal's label), the name in its colour with its crown and streak over the
first three flags it fought for (from its public card, `GetProgress`; the medal and the streak are its badges, by the
take's account, see [Badges](badges.md)); the name is
a button that opens that card. Under it, a footer strip: the flag it took
the tile for, with the country's name, and how long ago. The close is a round badge on the top-right corner,
outside the layout, so the card keeps `EDGE_PX` (12px) from the screen's edges. A
guest shows its name and the footer alone: a guest has no public card. A tap takes a tile, so a tap cannot ask: a hold is the one press a
tap never is, and the bomb's hold-to-drop is already one.

- **It names the missions the map draws on the tile** (`MissionsHere`, `missionsOnTile`): a strip above the footer,
  one line each, a small ring in `--mission` red, then "Mission:" (or "Flag mission:") and the mission's own words, as
  the Missions tab writes them. A mission counts here when the tile wears its red ring or stands on the ground it
  outlines; the peek brings the tile's ground and shields for it. None on a first visit, as the map draws none. See
  [Missions](missions.md).
- **Only past orbit.** In orbit a tile is too small to aim at, and the same
  gestures open a card on the territory instead: see [A territory, from
  orbit](territory-card.md).
- **It is drawn over the globe outside the four zones**, with the territory card,
  because it is about the tile under the pointer: below and right of a mouse, above a
  finger, which covers what is under it, turned and kept on screen at the edges
  (`domain/cardPlacement.ts`). A portal on the body, under the modals. Both cards are
  a `PointerCard`, which holds the placement, the close badge, Escape and the press
  outside.
- **It stays while the pointer is on the tile or on the card**, and goes
  `LEAVE_GRACE_MS` (300ms) after it left both, so a mouse can reach the name.
  On a phone it stays after the finger lifts. Its close button, Escape and a
  press anywhere outside it close it. **A tap that closes it does nothing else**
  (`swallowNextClick`), so closing the card never takes a tile; a mouse click
  goes through, so a painter's click still paints.
- `TilePeek` (`domain/tilePeek.ts`) holds both gestures over a spot, a tile here:
  `hover` starts a rest per spot, and a spot just clicked starts none; `begin`, `move` and `end` follow a
  finger, 12px of slack, a second finger cancelling it. The click that ends a
  hold is swallowed (`peeked`). `globe.ts` feeds it, hovers from a mouse alone,
  and calls `onPeek` with the pointer's place for a tile with a flag, and
  `onPeekEnd` when the mouse leaves that tile or the globe, or any press starts;
  nothing while a bomb is armed.
- **The card shown is a store**, `useTileHolder` (`app/players/tileHolderStore.ts`, see
  [Stores](stores.md)): the holder, whether the pointer is on its card (`hold`, `holderHeld`),
  and `dismiss`. `HolderCardPlace`, which `PlayScreen` renders with the backends, the globe's
  `pointing` and the account id, keeps it and shows `HolderCard` from it; `PlayScreen` reads none
  of it, and the place leaves nothing shown once it goes.
- `HolderCardPlace` asks `TileHolders.tileHolder` (`ClickService/GetTileHolder`), then
  `PlayerInfoBackend.playerInfo` (`PlayerService/GetPlayer` and
  `ProgressionService/GetProgress`, side by side), which answers a guest with its name
  alone: three calls, and planet never learns a name. Neither needs a
  token. A new peek aborts the last one. The lines it builds (`playerLineOf`,
  `deletedLineOf`) and `LEAVE_GRACE_MS` are in `app/players/tileHolder.ts`, which the
  territory card shares.
- **A take with no account is a deleted account** (`deletedLineOf`): the ledger
  keeps a deleted account's takes with no account, so the card says "Deleted
  account", the flag and when, with a name that opens nothing, and asks for no card.
  **So is a take whose account `GetPlayer` does not know** (`NotFound`): an account
  asked deleted is hidden for its 30 days of grace, and its takes keep its id until
  the purge.
- **The card names a player only while the tile wears the flag of its last take**
  (`holderShown`): the ledger is a second behind, and an operator's write keeps no
  take, so a tile that changed hands since names nobody rather than the wrong
  player. The take is of the map on screen: in an arena only the arena's, so a
  capital nobody took there says "No record of who took this tile."; after it,
  the world's again (the backend's [Arena battles](../../backend/docs/arenas.md#the-ledgers-readers)).
- **A card always opens; when it cannot name a player, it says why**
  (`HolderErrorCard`, `HolderError` in the store), over the tile's flag and its
  country: "No record of who took this tile." when no take of the tile's flag
  holds it (`unknown`), and "Who took this tile could not be loaded." when a read
  fails (`failed`, logged to the console too). A read the next peek aborted shows
  nothing.
- **A tile the player holds itself says "You took it"** (`HolderShown.mine`): the
  holder is the account `AccountStore` reads, which it reads for a signed-in
  player alone, so a guest sees its own tile under its guest name.
- The fake planet names one of the fake roster for each take of its bots
  (`FAKE_TAKERS`, a deleted account among them), and the player's own takes after an
  account it cannot name.
  In `npm run dev`, a hot reload can leave a card behind; reload the page.
