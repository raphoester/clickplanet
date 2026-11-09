# The promo module (`internal/promo/`)

**What the game shows of a battle before it starts: a teaser of cards, each a background and words over it.** The
cards are data, kept per rule set under the set's name, the way planet and the antibot keep their rules (the owner,
2026-11-05: "game modes and battle kinds should be represented as data"). A battle that plays `urf` is teased with
`urf`'s cards, whichever entry or season's finale plays it.

```
internal/promo/
  module.go                         Config (database, backgrounds), NewModule: the server's
  job.go                            JobConfig, ListDue, PostTeasers: the teasers job's (cmd/teasers), see teasers-job.md
  internal/
    teasers/                        a battle's teaser as a post: when it goes out, its words, the post through
                                    publishing's InternalService (see teasers-job.md)
    teaser_job/                     the teasers job's alone, which the server never builds (see teasers-job.md)
    migrations/                     backgrounds, cards
    placeholders/                   RulePaths, BattlePaths, Battle, Check, Fill: `{path}` in a card's words, filled with
                                    a value of planet.v1.Rules or of the battle
      rpc_scoring_points/           a battle's points by its times, from scoring.v1.InternalService
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
        rpc_gamemaster_battles/     the set and the times a battle plays, from gamemaster.v1.InternalService
      list_teasers_handler/teasers_query/   the battles to come with their cards, from gamemaster.v1's GetTimeline
      list_due_teasers_handler/due_teasers_query/   those within 36 hours no post teases, from publishing's ListPosts
      post_teaser_handler/          PostTeaser: the teaser's outcome on the wire
```

## A card

**A card is a background of the library, a headline and a line** (`cards.Draft`): a headline of 1 to 40 characters
and a line of 120 at most, both measured as written, and a set shows 0 to 5 of them, in order (`cards.NewDeck`, every
reason at once, `ErrInvalidCards`). **None is a set whose battles are not teased**: the client shows no teaser at all
with no cards (the owner: "If there is no carousel let's just not tease anything").

**Words name a rule's value with a placeholder**, `{path}`, the path of a field of `planet.v1.Rules` as protojson
writes it: `Clicks refill {clicks.tempo}× faster`, `A box every {bonuses.boxes.minInterval}`
(`placeholders.RulePaths`, `Check`). **Only a number or a duration**: a boolean, a list (the toll's steps) or a whole
message reads as no word in a sentence, and is refused. A brace that opens or closes no placeholder is refused too.

**Words name the battle's points too** (`placeholders.BattlePaths`): `{battle.times}`, the battle's points' times (1
to 10), and `{battle.points.first}`, `{battle.points.second}`, `{battle.points.third}`, what the first three ranks of
its round score. `1st place wins {battle.points.first} points` reads `75` in a ×3 battle and `50` in a ×2 one, with
nothing to edit. A set's cards follow every battle that plays it, so the words never hold the number. **The lengths
count the placeholder as written**: `1st place wins {battle.points.first} points` is 43 characters, a line and not a
headline.

**The values are read when a player reads the cards, never written into them** (`placeholders.Fill`): a number with
no trailing zero (`3`, `0.2`), a duration in its largest whole unit (`1 h`, `2 min`, `90 s`, `500 ms`, `0 s`). A save
of the set in planet, or of the battle's times in the schedule, shows on the next read, with no event and no lag. An earlier design rendered the words into the
images and re-rendered them on each rule change; the owner dropped it as over-engineered (2026-11-05).

**A set's cards are kept under its name**, and only for a set planet keeps (`save_cards_usecase` asks planet's
`GetRuleSetNames`, `ErrUnknownSet`). Cards kept for a set planet later deletes stay, read by nobody.

## The battle's points

**Scoring keeps the one table of points** (`books.points`: 25, 18, 15, 12, 10, 8, 6, 4, 2, 1, times the round's
`times`), and promo asks it (`scoring.v1.InternalService/GetRankPoints(times)`) rather than keeping a copy that would
drift the day scoring changes it. The gamemaster cannot answer the points: it only sends scoring the times, and scoring's
round does not exist before the battle starts, so `GetRound` cannot either. `GetRankPoints` is scoring's rule for any
times, not a read of a round.

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
asks the gamemaster what the battle plays (`gamemaster.v1.InternalService/GetBattle`: its set and its times), reads the
set's cards, asks planet what the set plays (`planet.v1.InternalService/GetRuleSet`) and scoring what each rank of a
round of those times scores (`scoring.v1.InternalService/GetRankPoints`), and fills them. It answers **no cards** for a
set with none, and **NotFound** for a battle the schedule does not hold. Never the set's name.

- **A card whose words no longer fill is left out**, logged at Error (`a teaser card does not fill`, with the set, the
  card and the battle), and the other cards still show: a rule renamed after the cards were written, or a value
  neither the set nor default holds. **A set planet no longer keeps teases nothing**, logged the same way.
- **A cached GET**: `NO_SIDE_EFFECTS`, so Connect sends it as an HTTP GET, and `NewCacheInterceptor` sets
  `Cache-Control: public, max-age=60, s-maxage=60`. A card, a rule or a battle's times changed reaches a page within a
  minute.

## The operator

`promo.v1.AdminService`, on the console and admin listeners (see [Operator tools](admin.md)), for the backoffice's
Promo page: `CreateBackgroundUpload`, `KeepBackground`, `ListBackgrounds`, `RemoveBackground` for the library;
**`GetCards`**, every set planet keeps with its cards as written and as players read them now (`filled`, or a
`problem` saying why not), and every placeholder of the rules with default's value as an example, for the page's
picker. **A set plays in no battle of its own**, so `GetCards` fills its cards as in an example battle of 3 times, the
Final Battle's (`example_battle`: its times, and each battle placeholder with its value there); and
**`SaveCards(rule_set, cards)`**, which replaces a set's cards (none removes them): `InvalidArgument` for cards the
rules refuse, `FailedPrecondition` for a set planet does not keep or a background not in the library.

## The teasers on TikTok and YouTube

**`promo.v1.AdminService/ListTeasers(within)` is each battle that starts within the window, by start, with its cards as
players read them now** (2026-11-09). Each `Teaser` is the id `GetTeaser` takes (the entry's, or the finale's round), the battle's start and end,
the schedule's words for it, its points' times, whether it is a season's Final Battle, and its cards, filled by the
teaser's own query (`teaser_query`). **A battle whose set has no card is left out**, as the game teases nothing of it,
and so is one the schedule dropped since its timeline was read. The battles come from the gamemaster's `GetTimeline`
(`teasers_query/rpc_gamemaster_timeline`): the ones the players see coming. `InvalidArgument` for a window of nothing,
or of more than a week. A read-only tool for Claude too (see [Claude as a connector](connector.md)).

**Promo posts each battle's teaser itself**, about a day before the battle, through the [publishing
module](publishing.md): a carousel of the cards on TikTok, a slideshow of them on YouTube. **`ListDueTeasers`** is the
battles within 36 hours that no post teases yet; **`PostTeaser(battle_id, photo_urls, video_url)`** checks the battle is
still due, writes the words, plans the slot and posts it, and answers posted, held, skipped, busy or failed. The
teasers job only renders the cards, which needs a browser, and uploads them. **The job is in this package too**, a
second root beside the module (`job.go`, `internal/teaser_job/`), as the clips job is in clips': its own binary,
`cmd/teasers`. See [Battle teasers on TikTok and YouTube](teasers-job.md).

## Kept

`promo` schema, migration `20261105120000_promo`: `backgrounds(background_id, name, media_type, size_bytes,
kept_at)` and `cards(rule_set, position, background_id, headline, line)`, keyed by the set and the position, with a
foreign key to the background. One pool, `promo.database`; the module runs nothing, so its pool is closed as a
closer, once the listeners shut down.

## Who it asks

**Promo depends on planet, the gamemaster, scoring and publishing, and nothing depends on promo.** It asks publishing
the posts that tease a battle (`ListPosts`, on each `ListDueTeasers` and `PostTeaser`), a teaser's slot (`PlanPost`
with `by`) and the post (`SchedulePost`), on its `InternalService` (see [Battle teasers on TikTok and
YouTube](teasers-job.md)). It asks planet the sets it
keeps (`GetRuleSetNames`, on each `SaveCards` and `GetCards`) and the values one plays (`GetRuleSet`, on each teaser
read and once per set with cards on each `GetCards`), the gamemaster what a battle plays (`GetBattle`, on each teaser
read) and the battles to come (`GetTimeline`, on each `ListTeasers`), and scoring what each rank of a round of its times scores (`GetRankPoints`, on each teaser read with cards, and
once for the example battle on each `GetCards`). The gamemaster stays unaware of promo: the client asks promo for a battle's cards when it shows the
teaser, rather than the gamemaster's timeline carrying them every second.

## Tests

`placeholders`' tests pin the paths (numbers and durations, no boolean, list or message; the battle's, which no rule
shares), the words filled (a number, a duration in each unit, a battle's points at ×3 and ×2), a value the set does not
play, a rank the battle does not score, and every refusal. `cards`' pin the deck (order, none, every
reason at once, characters and not bytes), `backgrounds`' the file rules, the name and the id. The use cases run on
the memory fakes: a background kept as the bucket holds it, a file never sent, a background a card shows that stays
with its file, the cards of a set planet does not keep. The stores and queries run on postgres: a background kept again
renamed, removed, refused with the sets named; a save that replaces a set's cards, none that removes them, a card on a
background not kept; the library newest first with its URLs; every set with its cards as written and as read in the ×3 example battle, its battle placeholders, a card
that does not fill, a set planet will not tell; a battle's cards filled (an entry, a finale), its own points (×3 and
×2), none, no battle, a card left out and logged, a set planet no longer keeps. `cache_interceptor_test.go` pins the GET and its header. The smoke
test `TestAnOperatorWritesASetsCardsAndAPageReadsTheBattlesTeaserFilled` gives planet `rush`, uploads a background
to the fake bucket, saves `rush`'s cards on the console listener, reads season 0's finale's teaser on the public
listener, filled with rush's values and the finale's points (`1st place wins 75 points`, ×3), and is refused the
background's removal.

The teasers' tests, the server's and the job's, are in [Battle teasers on TikTok and YouTube](teasers-job.md#tests).
