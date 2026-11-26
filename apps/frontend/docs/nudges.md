# Nudges

**A nudge is a full screen the game shows to remind or to ask, with no fact of the moment behind it**: the next
battle's teaser, the shop's sale while one runs, and the asks (a coffee, a TikTok follow, a YouTube subscribe). A moment (a title, a day closed, a
battle's start, a clip, a flag's mission) tells something that just happened and plays once; a nudge comes back on a
clock. Both go through the one announcements queue (see [Battles](battle.md#the-announcements-queue)), so nothing is
drawn over anything. Before 2026-11-12 the coffee was a popup of its own (`DonationModal`), drawn over the teaser.

## The lane

`Moments` (`app/announcements/Moments.tsx`) puts at most one nudge in the queue at a time:

- **Moments first.** A nudge is announced only when the queue is empty, and only once the map showed for
  `NUDGE_SETTLE_MS` (3 s, `useNudgeLane`), so the moments due as a page opens (a day's reveal, a clip) queue before it.
  A moment that arrives while a nudge shows waits behind it.
- **The teaser before any ask.** When both are due, the teaser goes in. Its own rules say when it is due (see
  [Battles](battle.md#the-teaser)): in the 24 hours before the battle, every 3 hours, and once more in its last hour.
  The asks never hold it back.
- **Then the sale, while one runs** (`saleNudgeDue`, `useSaleNudge`; see [The sale](#the-sale)). **No ask shows while a
  sale runs**: the sale takes their turn.
- **Then one ask** (`askDue` in `domain/nudges.ts`), when none of the asks and no teaser was closed in the last 12 hours
  (`ASK_SPACING_MS`). The asks take turns: the one closed longest ago, and the first time coffee, TikTok, YouTube.
- **No ask and no sale on a guided page** (a first visit, until its 50 takes, see [The front](front-guide.md)), or
  **during a battle**. The teaser keeps its own rule there: a guided page sees it after 30 clicks.

## The asks

`app/nudges/AskNudge.tsx`, a `Modal` each:

- **Coffee**: "Like ClickPlanet?", the logo, and Buy me a coffee (`BuyMeACoffee`).
- **TikTok**: "ClickPlanet on TikTok", and "Follow @clickplanetlol" to `TIKTOK_PROFILE`.
- **YouTube**: "ClickPlanet on YouTube", and "Subscribe" to `YOUTUBE_CHANNEL`.

**Close counts as asked.** **The link closes the screen and counts as followed**: a follow opened is never asked
again, a coffee opened not for 30 days. The browser keeps both times per ask in `clickplanet-nudges` (`askedOf`,
`askClosed`, `askOpened`); the teaser keeps its own in `clickplanet-battle-teased`.

## The sale

**`SaleNudge` (`app/nudges/SaleNudge.tsx`) is the shop's sale, black and gold** (Black Friday 2026, the owner's call on
2026-11-26; see [The shop](shop.md#a-sale)): the title reveal's stage on ink, a gold light behind it and a faint gold
pinstripe; "Black Friday" slammed in gold; the cut ("-50%") on a gold swing tag on its string, the gold rays turning
behind it; "On every plan for your region's flag"; the three plans, each its own price crossed out over the sale's in
gold (the sign in Rubik, `Money`); the clock to the second ("Ends in 2d 04h 12m 09s"); one red button, "Put your region
on the map", which opens what the shop's door opens (`useFlagForm.openShop()`: the account's flags, or a new one when it has none); and under it "Off the first payment. A month or a year
then renews at its full price." No sound, no shake, no image file. The close button (top right) and Escape close it.

- **Due** on the first visit while the sale runs, then once a day at most (`SALE_NUDGE_EVERY_MS`), and once more in its
  last 6 hours (`SALE_LAST_CALL_MS`). A sale is known by its end (`saleKeyOf`).
- **Close counts as seen**, and so does the button. The browser keeps the sale and the time in
  `clickplanet-sale-seen`.
- A sale that ends while it shows closes it.

**No count of its own.** Whether the asks work is read where it lands: the followers in Zernio's daily stats, the
coffees on Buy Me a Coffee.

## In fake mode

`startSale(5)` on the console: the sale's nudge comes at once on a page past its guided visit (see [The
shop](shop.md#in-fake-mode)).

`localStorage.removeItem("clickplanet-nudges")` and a reload: the coffee comes 3 s after the map (on a browser past
its guided visit). To see the next ask in turn, set the last one closed more than 12 hours back, for example
`localStorage.setItem("clickplanet-nudges", JSON.stringify({shown: {coffee: Date.now() - 13 * 3600e3}, opened: {}}))`.
