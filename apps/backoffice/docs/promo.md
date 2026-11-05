# Promo: the battle teasers

`src/promo/`, one page (`PromoPage`), in the menu under Ops, at `#/promo`. **A battle is teased in the game with its
rule set's cards**: a background and words over it, in a carousel the game opens before the battle. The backend's
promo module keeps them (`promo.v1.AdminService`, see the backend's [Promo](../../backend/docs/promo.md)): a library of
backgrounds, and up to five cards per rule set planet keeps, under the set's name. A set with no cards is not teased.

## The backgrounds

- **Upload** (`BackgroundsPanel`, `PickedImage`): "Pick images" takes one image or more (`image/jpeg`, `image/png`,
  `image/webp`, 2 MB at most). Each picked image has a name (the file name without its extension) and its own Upload
  button, three steps as a video's (see [Videos](videos.md)): `CreateBackgroundUpload`, one `PUT` straight to the
  bucket (`putFile`), then `KeepBackground`. A refused name keeps the file the bucket holds and calls `KeepBackground`
  alone; `KeepBackground`'s `failed_precondition` (the file is not in the bucket) sends it again. **An upload does not
  ask first**, as a video's does not.
- **The library** (`ListBackgrounds`, the newest first): each image cropped to the 3:4 card it is shown in, its name,
  its size and **Remove**, which asks first. **The server refuses to remove a background a card shows**, and the
  refusal names the sets: "Change the cards that show it first: …". The images load from their public URL on
  `media.clickplanet.lol` (in the CSP's `img-src`, see [Deploy](deploy.md)).

## The cards

- **A set at a time** (`CardsPanel`): the Rule set picker lists every set planet keeps (`GetCards`), each with its
  count of cards, and opens on the first that has some.
- **Each card** (`CardEditor`) is a background picked from the library, a headline of 1 to 40 characters and a line of
  120 at most, each counted as it is typed; ↑ and ↓ move it, Remove takes it out, and Add a card adds one, five at
  most.
- **A value of the rules is a placeholder**, `{path}` (`clicks.tempo`, `bonuses.boxes.minInterval`): "Insert a value at
  the cursor" lists every path `GetCards` answers, with default's value, and writes the one picked where the cursor
  was, in the headline or the line. The server fills it when a player reads the card, so the card follows the set's
  rules with nothing to edit.
- **So is a value of the battle**: `{battle.times}` and `{battle.points.first}`, `second`, `third`, the points the first
  three places win, listed in the picker's second group. Each battle fills its own (`75` in a ×3 battle, `50` in a ×2
  one). A set plays in no battle of its own, so the page shows them **as in the server's example battle**, a ×3 one
  (`GetCards`' `example_battle`): the group is named "The battle (example: ×3)", each value reads `(×3: 75)`, and a card
  that names one says "Example: a ×3 battle" under its preview. The lengths count a placeholder as written, so
  `1st place wins {battle.points.first} points` (43) goes in the line.
- **The preview is the card as players read it.** A card as it was saved shows what the server filled (`filled`), or
  the server's `problem` when it does not fill. A card being written is filled in the page (`fill.ts`, the server's
  rules: a number with no trailing zero, a duration in its largest whole unit), with the values the set plays: planet's
  `GetRules`, the set's own over default's (`playedBySet`, over the Game rules' `overlay`); and the battle's values as
  the server wrote them for the example battle.
- **Save asks first** ("Save the 3 cards of urf?", or "Remove every card of urf? Its battles are not teased then.")
  and sends the set's cards in their order (`SaveCards`), which replace what it had. It stays disabled until something
  changed and **while a card shows a problem the server would refuse**: no background, a headline or a line too long,
  a placeholder that names no number or duration, a value the set does not play, a brace alone. The server's own
  refusal is quoted.

## In the demo

`npm run demo`: three made-up backgrounds drawn in the page (SVG), `rush` with three cards (one names the battle's
points), an upload that the
made-up bucket takes in two seconds (an image only), and a removal refused while a card shows the background, as the
server does. The tests drive the page on `FakeAdmin.promo` (`FakePromo`), which lists the sets of planet's fake and
answers a ×3 example battle.
