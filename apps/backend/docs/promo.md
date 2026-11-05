# The promo module (`internal/promo/`)

**What the game shows of a battle before it starts: a teaser of cards, each a background and words over it.** The
cards are data, kept per rule set under the set's name, the way planet and the antibot keep their rules (the owner,
2026-11-05: "game modes and battle kinds should be represented as data"). A battle that plays `urf` is teased with
`urf`'s cards, whichever entry or season's finale plays it.

```
internal/promo/
  module.go                         Config (database, backgrounds), NewModule
  internal/
    migrations/                     backgrounds, cards
    placeholders/                   Paths, Check, Fill: `{path}` in a card's words, filled with a value of planet.v1.Rules
    backgrounds/                    BackgroundID, MediaType, File, Rules (FileError), Background (Keep, TakeSnapshot),
                                    BackgroundSnapshot (Restore); the IDProvider port; fakes behind the testing tag
      s3_background_bucket/         the files, in the bucket under promo/backgrounds/<id>: upload links, what a file
                                    uploaded is, its deletion, and its public URL
      postgres_background_store/    Keep (a background kept again takes its new name) and Remove, refused while a card
                                    shows it
      uuid_id_provider/             a random UUID per background
      usecases/create_upload_usecase/   draws an id, checks the type and the size, signs the upload link
      usecases/keep_background_usecase/ reads the file back from the bucket, keeps the background with what it holds
      usecases/remove_background_usecase/ the row, then the file
    cards/                          SetName, Draft, Card, Deck (NewDeck, TakeSnapshot), DeckSnapshot; fakes behind the
                                    testing tag
      postgres_card_store/          Save: a set's cards replaced, in a transaction
      rpc_planet_rules/             the sets planet keeps and the values each plays, from planet.v1.InternalService
      usecases/save_cards_usecase/  checks the cards and that planet keeps the set, then replaces its cards
    promov1controller/              AdminService and PromoService (bags), NewCacheInterceptor
      promoread/                    Filled: a card as players read it
      create_background_upload_handler/  keep_background_handler/  remove_background_handler/  save_cards_handler/
      list_backgrounds_handler/backgrounds_query/   PostgresQuery: the library, the newest first, with public URLs
      get_cards_handler/cards_query/   PostgresQuery: every set planet keeps, its cards as written and as read
      get_teaser_handler/teaser_query/ PostgresQuery: a battle's cards, filled
        rpc_gamemaster_battles/     the set a battle plays, from gamemaster.v1.InternalService
```

## A card

**A card is a background of the library, a headline and a line** (`cards.Draft`): a headline of 1 to 40 characters
and a line of 120 at most, both measured as written, and a set shows 0 to 5 of them, in order (`cards.NewDeck`, every
reason at once, `ErrInvalidCards`). **None is a set whose battles are not teased**: the client shows no teaser at all
with no cards (the owner: "If there is no carousel let's just not tease anything").

**Words name a rule's value with a placeholder**, `{path}`, the path of a field of `planet.v1.Rules` as protojson
writes it: `Clicks refill {clicks.tempo}× faster`, `A box every {bonuses.boxes.minInterval}`
(`placeholders.Paths`, `Check`). **Only a number or a duration**: a boolean, a list (the toll's steps) or a whole
message reads as no word in a sentence, and is refused. A brace that opens or closes no placeholder is refused too.

**The values are read when a player reads the cards, never written into them** (`placeholders.Fill`): a number with
no trailing zero (`3`, `0.2`), a duration in its largest whole unit (`1 h`, `2 min`, `90 s`, `500 ms`, `0 s`). A save
of the set in planet shows on the next read, with no event and no lag. An earlier design rendered the words into the
images and re-rendered them on each rule change; the owner dropped it as over-engineered (2026-11-05).

**A set's cards are kept under its name**, and only for a set planet keeps (`save_cards_usecase` asks planet's
`GetRuleSetNames`, `ErrUnknownSet`). Cards kept for a set planet later deletes stay, read by nobody.

## The backgrounds

**A library of images every set's cards pick from**, uploaded on the backoffice's Promo page, the clip videos' way
(see [Clip videos](clip-videos.md)): `CreateBackgroundUpload(content_type, size_bytes)` answers an id and a link, the
page sends the file there with one `PUT`, then `KeepBackground(background_id, name)` reads it back with a `HEAD` and
keeps the type and the size the bucket holds. **A JPEG, a PNG or a WebP of 2 MiB at most**
(`promo.backgrounds.maxBytes`), and a name of 1 to 100 characters. A card shows it cropped to a 3:4 portrait frame.

- **In the support module's bucket, `clickplanet-media`, under `promo/backgrounds/<id>`**, with the same key. **Public
  on `media.clickplanet.lol`**, which Cloudflare serves for `/promo/*` alone, cached a year: an id is drawn once and
  its file never changes, so a URL can be kept for good. The support files and the clip videos at the same bucket stay
  private (see `deploy/vps/README.md`, "Battle teasers"). `ListBackgrounds` and every card answer that public URL,
  never a signed link.
- **A background a card shows is not removed** (`ErrShown`, `FailedPrecondition`, naming the sets): the operator
  changes those cards first. The cards' foreign key holds it too.
- **A removal deletes the row, then the file**, so no card ever shows a file the bucket no longer holds. A file the
  bucket then will not delete stays there, listed nowhere; Cloudflare may serve a removed file from its cache for a
  while.
- **An upload never kept leaves its file in the bucket**, as a clip video's does.
- **Off** (`promo.backgrounds.enabled: false`), every background call is `FailedPrecondition` and the bucket block is
  not read. With no background no card can be saved, so no battle is teased.

## The teaser

**`promo.v1.PromoService/GetTeaser(battle_id)`, on the public router, is a battle's cards as players read them**: the
id the gamemaster's timeline names the battle by, its entry's id or `season-N/finale` for a season's finale. The query
asks the gamemaster which set the battle plays (`gamemaster.v1.InternalService/GetBattleSet`), reads the set's cards,
asks planet what the set plays (`planet.v1.InternalService/GetRuleSet`) and fills them. It answers **no cards** for a
set with none, and **NotFound** for a battle the schedule does not hold. Never the set's name.

- **A card whose words no longer fill is left out**, logged at Error (`a teaser card does not fill`, with the set, the
  card and the battle), and the other cards still show: a rule renamed after the cards were written, or a value
  neither the set nor default holds. **A set planet no longer keeps teases nothing**, logged the same way.
- **A cached GET**: `NO_SIDE_EFFECTS`, so Connect sends it as an HTTP GET, and `NewCacheInterceptor` sets
  `Cache-Control: public, max-age=60, s-maxage=60`. A card or a rule changed reaches a page within a minute.

## The operator

`promo.v1.AdminService`, on the console and admin listeners (see [Operator tools](admin.md)), for the backoffice's
Promo page: `CreateBackgroundUpload`, `KeepBackground`, `ListBackgrounds`, `RemoveBackground` for the library;
**`GetCards`**, every set planet keeps with its cards as written and as players read them now (`filled`, or a
`problem` saying why not), and every placeholder with default's value as an example, for the page's picker; and
**`SaveCards(rule_set, cards)`**, which replaces a set's cards (none removes them): `InvalidArgument` for cards the
rules refuse, `FailedPrecondition` for a set planet does not keep or a background not in the library.

## Kept

`promo` schema, migration `20261105120000_promo`: `backgrounds(background_id, name, media_type, size_bytes,
kept_at)` and `cards(rule_set, position, background_id, headline, line)`, keyed by the set and the position, with a
foreign key to the background. One pool, `promo.database`; the module runs nothing, so its pool is closed as a
closer, once the listeners shut down.

## Who it asks

**Promo depends on planet and the gamemaster, and nothing depends on promo.** It asks planet the sets it keeps
(`GetRuleSetNames`, on each `SaveCards` and `GetCards`) and the values one plays (`GetRuleSet`, on each teaser read
and once per set with cards on each `GetCards`), and the gamemaster which set a battle plays (`GetBattleSet`, on each
teaser read). The gamemaster stays unaware of promo: the client asks promo for a battle's cards when it shows the
teaser, rather than the gamemaster's timeline carrying them every second.

## Tests

`placeholders`' tests pin the paths (numbers and durations, no boolean, list or message), the words filled (a number,
a duration in each unit), a value the set does not play, and every refusal. `cards`' pin the deck (order, none, every
reason at once, characters and not bytes), `backgrounds`' the file rules, the name and the id. The use cases run on
the memory fakes: a background kept as the bucket holds it, a file never sent, a background a card shows that stays
with its file, the cards of a set planet does not keep. The stores and queries run on postgres: a background kept again
renamed, removed, refused with the sets named; a save that replaces a set's cards, none that removes them, a card on a
background not kept; the library newest first with its URLs; every set with its cards as written and as read, a card
that does not fill, a set planet will not tell; a battle's cards filled (an entry, a finale), none, no battle, a card
left out and logged, a set planet no longer keeps. `cache_interceptor_test.go` pins the GET and its header. The smoke
test `TestAnOperatorWritesASetsCardsAndAPageReadsTheBattlesTeaserFilled` gives planet `rush`, uploads a background
to the fake bucket, saves `rush`'s cards on the console listener, reads season 0's finale's teaser on the public
listener, filled, and is refused the background's removal.
