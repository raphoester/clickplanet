# Who holds a tile

**A mouse resting on a tile for half a second, or a finger holding one still for
half a second, opens a card beside the pointer on the player whose take holds
it** (`HolderCard`). One row: the medal of the title it wears (its name only as
the medal's label), the name in its colour with its crown and streak over the
first three flags it fought for (from its public card, `GetPlayer`), and a round
profile button that opens that card. Under it, a footer strip: the flag it took
the tile for and how long ago. The close is a round badge on the top-right corner,
outside the layout, so the card keeps `EDGE_PX` (12px) from the screen's edges. A
guest shows its name and the footer alone: a guest has no public card. A tap takes a tile, so a tap cannot ask: a hold is the one press a
tap never is, and the bomb's hold-to-drop is already one.

- **It is the one thing drawn over the globe outside the four zones**, because it
  is about the tile under the pointer: below and right of a mouse, above a
  finger, which covers what is under it, turned and kept on screen at the edges
  (`domain/cardPlacement.ts`). A portal on the body, under the modals.
- **It stays while the pointer is on the tile or on the card**, and goes
  `LEAVE_GRACE_MS` (300ms) after it left both, so a mouse can reach the button.
  On a phone it stays after the finger lifts. Its close button, Escape and a
  press anywhere outside it close it. **A tap that closes it does nothing else**
  (`swallowNextClick`), so closing the card never takes a tile; a mouse click
  goes through, so a painter's click still paints.
- `TilePeek` (`domain/tilePeek.ts`) holds both gestures: `hover` starts a rest per
  tile, and a tile just clicked starts none; `begin`, `move` and `end` follow a
  finger, 12px of slack, a second finger cancelling it. The click that ends a
  hold is swallowed (`peeked`). `globe.ts` feeds it, hovers from a mouse alone,
  and calls `onPeek` with the pointer's place for a tile with a flag, and
  `onPeekEnd` when the mouse leaves that tile or the globe, or any press starts;
  nothing while a bomb is armed.
- `useTileHolder` asks `TileHolders.tileHolder` (`ClickService/GetTileHolder`), then
  `PlayerInfoBackend.playerInfo` (`PlayerService/GetPlayer`), which answers a guest
  with its name alone: two calls, and planet never learns a name. Neither needs a
  token. A new peek aborts the last one.
- **A take with no account is a deleted account** (`DELETED_ACCOUNT`): the ledger
  keeps a deleted account's takes with no account, so the card says "Deleted
  account", the flag and when, with no profile button, and asks `GetPlayer` nothing.
- **The card shows only while the tile wears the flag of its last take**
  (`holderShown`): the ledger is a second behind, and an operator's write keeps no
  take, so a tile that changed hands since says nothing rather than the wrong
  player.
- **A tile the player holds itself says "You took it"** (`HolderShown.mine`): the
  holder is the account `AccountStore` reads, which it reads for a signed-in
  player alone, so a guest sees its own tile under its guest name.
- The fake planet names one of the fake roster for each take of its bots
  (`FAKE_TAKERS`, a deleted account among them), and the player's own takes after an
  account it cannot name.
  In `npm run dev`, a hot reload can leave a card behind; reload the page.
