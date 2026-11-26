# The shop

**A player pays for a regional flag's time on Stripe's own page: a new flag on the paywall after it made and sent it,
a flag it holds or one whose time ended (a claim) on that flag's pay page** (step 4 of the regional flags, 2026-11-24;
the flag made first, the review started at Next and the paywall after it, the owner, 2026-11-26; whoever pays holds the
flag, the owner, 2026-11-26): each page shows the plans once, asks one consent, and goes to the page `CreateCheckout`
answers. The
time paid is held, and starts when the team approves the flag, which then opens at once; an approved flag opens as it is
paid; a rejected flag is refunded in full by the server, its renewal stopped. Nothing about a card is typed in the
game. The backend's side is `shop.v1.ShopService` (`proto/shop/v1/shop.proto`): `ListPlans`, `CreateCheckout`,
`ListMyRenewals`, `StopRenewal`.

- **The plans**: a week paid once, a month and a year that renew, prices with VAT, in the currency's smallest unit
  (`PlanPrice.amount`, 400 is €4). The page prices each with `Intl.NumberFormat` in its language, a whole amount with no
  cents ("€4", "€10 / month").
- **A sale**, while one runs by the server's clock: `ListPlans` answers it beside the plans (`Sale`, its percent off and
  its end), and the page takes its cut off the prices (see [A sale](#a-sale)).
- **The shop off on the server** answers no plan: then no plan, no pay button, no shop door, no Add time and no renewal line
  shows, and the flag's paywall reads "Sent on …" and "Approved! It opens soon." as before. A server with no shop at all
  (`Unimplemented`) reads the same.

## Its doors

**The shop's door opens the region's flag's one flow** (see [Flag submissions](flag-submissions.md#the-doors)): Your
flags, the flags the account holds and sends, or step 1, "make your flag", with no price, for an account with none.
**The plans show on one page only, never in a list**: a new flag's paywall, or a flag's pay page. Until 2026-11-26 the
door opened a shop's modal that listed the plans and sent the player to the flag form, which listed them again with
Pay: the player picked a plan twice; then, that day, the plans showed inside the profile's list for Add time.

- **On a phone**, a bag at the end of the status bar's lead (`status-bar-shop`), so held sideways it is in the left pill;
  **on the desktop**, the rail's Shop, under its line above You (`menu-rail-shop` in `menu-rail-modals`). Each is a gold
  button, as the money comes from there, and a band of gold light crosses it every 5 seconds (`shop-shine`, an endless
  animation, see [Styling](styling.md)); none under reduced motion. Each calls `useFlagForm.open()`. The flag picker's
  row is gold with the same shine.
- **No door while the shop is off**: the plans are read at load for every page, a guest's too (`ListPlans` is a GET,
  the same answer for all), and with none the bag and the rail's Shop do not show.
- **The head of step 1** (`FlagPitch`) is the shop's pitch: the gold coin, "Put your region on the map" and three perks.

## The code

- `domain/shop.ts`: `Plan`, `PlanPrice`, `Sale`, `Catalog` (the plans and the sale), `CheckoutTarget` (a submission or
  a flag), `Renewal`; `saleRuns` and `salePriceOf` (a plan's first payment during a sale: half of 400, 1000 and 6000 is
  200, 500 and 3000); `payable` (a new flag not
  paid: sent, or approved), `awaitsPayment`, `flagRenewal` (the renewal on a flag or on the submission that opened it)
  and `flagTimeOf`, the facts a held flag's line says.
- `backends/shop.ts`: `Shop` and `ShopRefusedError` (`not-yours`, `not-found`, `unavailable`, `invalid`, the codes
  `PermissionDenied`, `NotFound`, `FailedPrecondition`, `InvalidArgument`). `ConnectShop` (`shopBackend.ts`) sends the
  token that names the account (`identity()`, as `ListMySubmissions`), `ListPlans` none; the reads are retried, and a
  checkout is not: each call is a new page on Stripe. `FakeShop` in fake mode.
- `app/shop/shopStore.ts`: `useShop`, the plans, the sale and the renewals; the sale is dropped at its end by a timer,
  with no read. `useShopFeed`, in `PlayScreen`, reads the plans at
  load for every page, and a signed-in account's renewals each time the flag's flow or Your flags opens (`refresh`,
  which reads the plans again too).
- `domain/shop.ts` also prices the plans against each other: `perDayOf`, `savingOf`, `bestValueOf`, `pickedOf`.
- `app/shop/PlanCards.tsx`: the plans side by side. `app/shop/PlanOffer.tsx`: the plans, the consent and Pay, on the
  flag's paywall and for Add time. `app/shop/prices.ts` and `Money.tsx`: a plan's name, cadence, price by the day,
  saving, and its price with the sign apart. `app/shop/flagTime.ts`: a founded flag's time line.
- `app/shop/shopReturn.ts`: the way back from Stripe's page.
- `Backends.toCheckout` goes to Stripe's page: `location.assign` in production.

## The offer

**The plans, the consent and Pay are one piece** (`PlanOffer`), on the flag's paywall (target the submission; see
[Flag submissions](flag-submissions.md#step-2-the-paywall)) and on a flag's pay page, for time added to a flag held or
a claim (target the flag; see [A flag's pay page](flag-submissions.md#a-flags-pay-page)):

- **The plans side by side** (`PlanCards`): three cards, a radio group. Each has its name, its price large (the sign in
  Rubik: Luckiest Guy's euro reads as a "<"), "Paid once" or "Every month", its price by the day ("€0.17 / day",
  rounded up so it is never told cheaper), and what it saves by the day against the dearest plan, rounded down ("-42%",
  "-71%", none on the week). The plan cheapest by the day wears "Best value" and is picked first.
- **The pick is kept** (`useShop.chosen`) for the session: the next offer picks it.
- **Colors** in their jobs: blue for the plan picked (a toggled button), gold for value (the savings, Best value), red
  only for the one thing to press.
- **One checkbox, unchecked** (`Consent`, `PayTerms.tsx`): "I want my flag to start as soon as it's approved. I know I
  can't cancel once it starts." This is the EU right of withdrawal's waiver for digital content: the server refuses a
  checkout without `starts_now`, and the page sends it only once the box is checked.
- **Pay €60**, the plan's price in the button, off until the box is checked. It calls
  `CreateCheckout{plan, submission_id or flag, starts_now: true}` and goes to its `url`. A refusal is one line under the
  button, the plan and the box stay, and Pay is on again; any other failure is "The payment page did not open. Try
  again."
- Under it (`FinePrint`), "Secure payment by Stripe", with "· Stop anytime" for a plan that renews, and the links to the
  terms of sale and the refunds (`/sale`, `/refunds`), in a new tab.
- **Back beside Pay**: on the paywall while the flag waits unpaid (a change of it), and on a flag's pay page opened from
  Your flags (back to it). Each widens to the plans' width (`shop-modal`).

## A sale

**While a sale runs, every price the shop shows is the sale's, and the game says how long it lasts** (Black Friday 2026,
50% off from Thu 26 Nov 17:15 UTC (planned for Fri 27 Nov 09:00 UTC, started at once by the owner) until Mon 30 Nov 22:59 UTC; the backend's [A sale](../../backend/docs/shop.md#a-sale)).
`ListPlans` answers the plans at their full price and the sale's raw facts (`percent_off`, `ends_at`); a page of an
older build ignores them and shows the full prices, which Stripe's page then lowers. The store drops the sale at its end
by itself, so the tags, the clock and the nudge go at the second it ends, with no read. A page loaded before the sale
starts sees it at its next read of the plans: a load, or the flag's flow or Your flags opened.

- **The offer** (`PlanCards`, `PlanOffer`, on the paywall and on a flag's pay page): over the cards, the sale's clock
  (`SaleClock`, "Black Friday · ends in 2d 04h 12m 09s", the game's countdown, to the second, never cut in two); each
  card's own price crossed out over the sale's price, large in gold (the line crosses the digits alone and the old price
  is faint, `Money`'s `struck`: through the euro's two bars it read as "£4"); "Paid once", or for a plan that renews "Then €10 a
  month", as the coupon is taken off the first payment only; the sale's cut ("-50%") in the gold of value, in the
  saving's place; the price by the day of the sale's price. Pay says the sale's price ("Pay €30"). The blue of the plan
  picked and the red of Pay keep their jobs.
- **The doors** (`status-bar-shop`, `menu-rail-shop`) wear the cut (`SaleTag`, "-50%"), gold on a dark well, outside
  the button (`shop-door`), which clips its shine: on the rail at its top corner, on the phone's status bar from the
  bag's lower corner, as the bar has no room above.
- **The nudge**: the sale full screen in the nudge lane (see [Nudges](nudges.md#the-sale)).

## Time for a flag held

**Your flags says each held flag's time while the shop sells** (`FlagHub`, `flagTimeOf`; see
[Flag submissions](flag-submissions.md#your-flags)):

- **A renewal that runs**, on the flag or on the submission that opened it: "Renews on Dec 24, 2026" and Stop renewal.
  `StopRenewal{renewal_id}` answers the renewal stopping, which replaces it in the store: "Ends on Dec 24, 2026". A stop
  that fails is one line under the list. The server refuses a second plan while a renewal runs: it would charge twice,
  so no Add time shows.
- **A renewal that stops**: "Ends on …" and Add time: it charges no more, so more time is no second charge.
- **No renewal**, the flag open: "Ends on …" (`Sold.endsAt`) and Add time; closed, however long ago (`Sold.closedAt`):
  "Closed on …" and Add time, until another player claims it.
- **Removed** (`Sold.removed`), shop on or off: "Removed on …", the operator's note, and nothing to pay.
- **Add time** opens the flag's pay page, target `flag`, and Back to Your flags.

The dates are each language's medium date, as "Sent on …".

## Back from Stripe

**Stripe sends the buyer back to `/play?shop=paid`, or `/play?shop=cancelled`** after leaving its page.
`main.tsx` reads `shop` (`domain/shopReturn.ts`) and drops it with `history.replaceState`, keeping the rest of the query
and the hash, and hands it to `PlayScreen` (`shopReturn`), which calls `returnFromShop` once:

- **The news line** (`ShopNews`, the news store's `shop` kind, 4 seconds): "Payment received!", or "Payment
  cancelled.". The flag's flow is closed; nothing else opens.
- **After a payment, the submissions are read every 3 seconds for 2 minutes at most**, and no more once a read finds
  none sent or approved and unpaid (`awaitsPayment`): the server marks a flag sent paid, or opens an approved one, from
  Stripe's webhook a moment after the buyer is back. An opening is then told as any other: "Bavaria is in the game!". Time added to a flag needs no read: the
  flags stream tells the flag's new end.

## In fake mode

`FakeShop` (`backends/fakeShop.ts`) sells the three plans (400, 1000 and 6000 eur) and refuses as the server does: a
guest, no consent, the shop off, a submission paid, rejected or unknown, an open flag the account does not hold, a
removed flag, a flag whose renewal runs (the account's, or another player's: `renewingElsewhere(code)`), and a target
already paid and not landed yet. A checkout answers
`<origin>/play?shop=paid`; `toCheckout` in fake mode comes back in the page (`returnFromShop`), since a load would forget
the fakes. Two seconds later, as the webhook: a flag sent is paid (`FakeFlagSubmissions.hold`, its time held), an
approved one opens (`openApproved`, a week, a month or a year), and a flag gets its time or is claimed
(`FakeFlagsBackend.pay`: the account holds it from then). A month or a year adds a renewal, which Stop renewal stops. `approveSubmission()` opens a
paid flag at once with its time, as the server's release; `rejectSubmission()` of a paid one makes it unpaid, as the
refund. A paid flag is not replaced.

Sign in (`signIn()`), make a flag from the shop or the flag picker, press Next, and pay on the paywall: the flag waits,
paid. Then `approveSubmission()`; or give the verdict while the paywall shows, and pay the approved flag, which opens two
seconds later. On the console:

- `shopOff()` / `shopOn()`: no plan, as a server with the shop off, or the three again.
- `refuseCheckouts("unavailable")`: the next checkouts are refused with that refusal (`not-yours`, `not-found`,
  `unavailable`, `invalid`); `refuseCheckouts()` lets them through again.
- `renewingElsewhere(code)`: another player's renewal still runs on the flag, as while Stripe tries a card again: a
  claim is refused, "You can't pay for this flag now."; `renewingElsewhere(code, false)` lets it through.
- `startSale(minutes = 60, percentOff = 50)`: the fake answers a sale that ends in that many minutes, and the page reads
  the plans again: the clock, the tags and the nudge show (the nudge once per sale: a new `startSale` is a new sale).
  `endSale()`: no sale, the plans read again. The fake has no sale until then.

To claim a flag: `endFlag("xb")` (Brittany's time ends; nobody holds it), then search "brit" in the flag picker:
"Brittany is free. Claim it." opens its pay page; pay, and two seconds later Brittany is open and in Your flags.

## The legal pages

`/sale` (the terms of sale) and `/refunds` are plain pages like the terms of service (see [Pages and routes](routes.md)).
The terms of service link them in "Regional flags", the privacy policy says what Stripe and we keep in "Payments", and the
home page's footer and the About modal list them beside the other two. Their text, and the privacy policy's Payments
section, carry `<!-- DRAFT: the owner reviews this text before the merge. -->`: the owner reviews it before the merge.
