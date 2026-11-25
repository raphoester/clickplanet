# The shop

**The founder of a regional flag pays for its time on Stripe's own page, as it sends the flag** (step 4 of the regional
flags, 2026-11-24; pay first, the owner the same day: "make the guy PAY FIRST NOW or he might change his mind"): the
page shows the plans, asks one consent, and goes to the page `CreateCheckout` answers. The time paid is held, and starts
when the team approves the flag, which then opens at once; a rejected flag is refunded in full by the server, its
renewal stopped, and only a paid flag reaches the review. Nothing about a
card is typed in the game. The backend's side is `shop.v1.ShopService` (`proto/shop/v1/shop.proto`): `ListPlans`,
`CreateCheckout`, `ListMyRenewals`, `StopRenewal`.

- **The plans**: a week paid once, a month and a year that renew, prices with VAT, in the currency's smallest unit
  (`PlanPrice.amount`, 400 is €4). The page prices each with `Intl.NumberFormat` in its language, a whole amount with no
  cents ("€4", "€10 / month").
- **The shop off on the server** answers no plan: then no plan, no pay button, no Add time and no renewal line shows,
  and an approval reads "Approved! It opens soon." as before. A server with no shop at all (`Unimplemented`) reads the
  same.

## The shop's modal

**The shop is where the money comes from, so it is a purchase path, not a price list** (the owner, 2026-11-24):
`ShopModal`, ready for more products. Today one, the region's flag, and what it shows follows the account
(`offerOf` in `domain/shop.ts`):

| The account | Its head | Then |
|---|---|---|
| no flag to pay for (a guest too) | a gold coin, "Your region's flag", "Put your region on the map", three perks | the steps at 1, the plans, **Add your flag** (the flag's place: a guest signs in, a player sends its flag and pays in the same form) |
| a flag sent, not paid (its payment page closed) | its name | the steps at 2, then the plans, the consent and **Pay €60**: it pays from the shop |
| a flag paid, in review | its name, the Waiting chip | the steps at 3, "Paid! We check your flag, then it opens.", **Your flag** |
| a flag paid and approved, the moment before it opens | its picture and name, the Approved chip | every step done, "Approved! It opens now.", **Your flag** |
| a flag approved and not paid (refunded while it waited, or approved before pay first) | its picture and name, the Approved chip | the plans, the consent and Pay, with no steps |
| a flag founded, open or closed less than 30 days ago, or its renewal stopping | its flag, "Ends on …" or "Closed on …" | the plans, the consent and Pay, for the flag; **New flag** under them |
| a flag founded whose renewal runs | its flag, "Renews on …" | **Your flag**: no second plan, it would charge twice; **New flag** under it |

- **The steps** ("Send your flag", "Pay", "We check it"): three coins on a line, gold up to where the flag is, a check
  on each one done. Only on the way to a new flag.
- **New flag** for a founder (the modal's footer, secondary: Pay stays the one thing to press): it closes the shop and
  opens the flag form empty (`useFlagForm.compose`), as the footer's New flag in the flag's place. Without it, a founder
  whose flag closed found no way to a second flag in the shop for 30 days. The server takes a new flag whenever none
  waits, and `offerOf` gives these two offers only then.
- **The plans side by side** (`PlanCards`): three cards, a radio group. Each has its name, its price large (the sign in
  Rubik: Luckiest Guy's euro reads as a "<"), "Paid once" or "Every month", its price by the day ("€0.17 / day",
  rounded up so it is never told cheaper), and what it saves by the day against the dearest plan, rounded down ("-42%",
  "-71%", none on the week). The plan cheapest by the day wears "Best value" and is picked first.
- **The pick is kept** (`useShop.chosen`) for the session: a plan picked in the shop is the one the flag form picks.
- **Colors** in their jobs: blue for the plan picked (a toggled button), gold for value (the savings, Best value, the
  steps), red only for the one thing to press.
- **Its doors**: on a phone, a bag at the end of the status bar's lead (`status-bar-shop`), so held sideways it is in
  the left pill; on the desktop, the rail's Shop, under its line above You (`menu-rail-shop` in `menu-rail-modals`).
  Each is a gold button, as the money comes from there, and a band of gold light crosses it every
  5 seconds (`shop-shine`, an endless animation, see [Styling](styling.md)); none under reduced motion. Each opens it with
  `useScreen.setShopOpen(true)` (see [The screen](screen.md)). The shop reads the submissions and the renewals again as it
  opens.
- **No door while the shop is off**: the plans are read at load for every page, a guest's too (`ListPlans` is a GET,
  the same answer for all), and with none the bag and the rail's Shop do not show.

## The code

- `domain/shop.ts`: `Plan`, `PlanPrice`, `CheckoutTarget` (a submission or a flag), `Renewal`; `payable` (a new flag not
  paid: sent, or approved), `awaitsPayment`, `flagRenewal` (the renewal on a flag or on the submission that opened it), `canAddTime` (open,
  or closed less than 30 days ago) and `flagTimeOf`, the facts a founded flag's line says.
- `backends/shop.ts`: `Shop` and `ShopRefusedError` (`not-yours`, `not-found`, `unavailable`, `invalid`, the codes
  `PermissionDenied`, `NotFound`, `FailedPrecondition`, `InvalidArgument`). `ConnectShop` (`shopBackend.ts`) sends the
  token that names the account (`identity()`, as `ListMySubmissions`), `ListPlans` none; the reads are retried, and a
  checkout is not: each call is a new page on Stripe. `FakeShop` in fake mode.
- `app/shop/shopStore.ts`: `useShop`, the plans and the renewals. `useShopFeed`, in `PlayScreen`, reads the plans at
  load for every page, and a signed-in account's renewals each time the flag's place opens (`refresh`, which reads the
  plans again too).
- `domain/shop.ts` also prices the plans against each other: `perDayOf`, `savingOf`, `bestValueOf`, `pickedOf`; and
  `offerOf`, what the shop sells the account now.
- `app/shop/ShopModal.tsx`: the shop's modal. `app/shop/PlanCards.tsx`: the plans side by side.
  `app/shop/PlanOffer.tsx`: the plans, the consent and Pay, in the shop and in the flag's place. `app/shop/prices.ts`
  and `Money.tsx`: a plan's name, cadence, price by the day, saving, and its price with the sign apart.
  `app/shop/flagTime.ts`: a founded flag's time line.
- `app/shop/shopReturn.ts`: the way back from Stripe's page.
- `Backends.toCheckout` goes to Stripe's page: `location.assign` in production.

## Pay as the flag is sent

**While the shop sells, the flag form ends with the plans, the consent and "Pay €60"** (`FlagSubmissionForm`), in place
of Send: the plan picked in the shop is picked here, and Pay is off until the box is checked. It uploads the picture,
sends the flag, then calls `CreateCheckout{plan, submission_id, starts_now: true}` for the submission just sent and goes
to its `url`. A checkout that fails after the send puts the form away (`useFlagForm.sent(payFailure)`): the place shows
the flag sent, "Sent! Pay to send it for review.", the plans and the failure line under Pay, so the player pays again
there. While the shop is off, the form sends as before.

- **A flag sent and not paid** may still change (the footer's Change), and offers the plans.
- **A flag paid** reads "Paid! We check your flag, then it opens." (`Submission.paid`, from the wire's `paid`): no plan,
  no Change (the server refuses to replace a paid flag) and no New flag, since one waits.
- **Approved and paid**: "Approved! It opens now.", the moment before the flags stream tells it open.

## The offer

**The plans, the consent and Pay are one screen** (`PlanOffer`), in the shop and in the flag's place, for a flag sent
and not paid, a flag approved and not paid ("Approved! Pick a plan to open it."), and time for a founded flag:

- **The plans side by side**, the best value picked first, or the plan the player picked last.
- **One checkbox, unchecked** (`Consent`, `PayTerms.tsx`): "I want my flag to start as soon as it's approved. I know I
  can't cancel once it starts." This is the EU right of withdrawal's waiver for digital content: the server refuses a
  checkout without `starts_now`, and the page sends it only once the box is checked.
- **Pay €60**, the plan's price in the button, off until the box is checked. It calls
  `CreateCheckout{plan, submission_id or flag, starts_now: true}` and goes to its `url`. A refusal is one line under the
  button, the plan and the box stay, and Pay is on again; any other failure is "The payment page did not open. Try
  again."
- Under it (`FinePrint`), "Secure payment by Stripe", with "· Stop anytime" for a plan that renews, and the links to the
  terms of sale and the refunds (`/sale`, `/refunds`), in a new tab.
- In the flag's place, Back beside Pay when it adds time, and the footer's "New flag" turns secondary: Pay is the one
  thing to press. The place and the form widen to the shop's width while they offer the plans.

## Time for a founded flag

**"Your flags" says each founded flag's time while the shop sells** (`FoundedFlags`, `flagTimeOf`):

- **A renewal that runs**, on the flag or on the submission that opened it: "Renews on Dec 24, 2026" and Stop renewal.
  `StopRenewal{renewal_id}` answers the renewal stopping, which replaces it in the store: "Ends on Dec 24, 2026". A stop
  that fails is one line under the list. The server refuses a second plan while a renewal runs: it would charge twice.
- **A renewal that stops**: "Ends on …" and Add time: it charges no more, so more time is no second charge.
- **No renewal**, the flag open: "Ends on …" (`Sold.endsAt`) and Add time; closed less than 30 days ago (`Sold.closedAt`):
  "Closed on …" and Add time. Closed longer: nothing.
- **Add time** shows the flag with the same plans in the place, target `flag`, and Back to the list.

The dates are each language's medium date, as "Sent on …".

## Back from Stripe

**Stripe sends the buyer back to `/play?shop=paid`, or `/play?shop=cancelled`** after leaving its page.
`main.tsx` reads `shop` (`domain/shopReturn.ts`) and drops it with `history.replaceState`, keeping the rest of the query
and the hash, and hands it to `PlayScreen` (`shopReturn`), which calls `returnFromShop` once:

- **The news line** (`ShopNews`, the news store's `shop` kind, 4 seconds): "Payment received!", or "Payment
  cancelled.". The flag's place and the shop are closed; nothing else opens.
- **After a payment, the submissions are read every 3 seconds for 2 minutes at most**, and no more once a read finds
  none sent or approved and unpaid (`awaitsPayment`): the server marks a flag sent paid, or opens an approved one, from
  Stripe's webhook a moment after the buyer is back. An opening is then told as any other: "Bavaria is in the game!". Time added to a flag needs no read: the
  flags stream tells the flag's new end.

## In fake mode

`FakeShop` (`backends/fakeShop.ts`) sells the three plans (400, 1000 and 6000 eur) and refuses as the server does: a
guest, no consent, the shop off, a submission paid, rejected or unknown, a flag the account did not found, closed 30 days
ago or more, or whose renewal runs, and a target already paid and not landed yet. A checkout answers
`<origin>/play?shop=paid`; `toCheckout` in fake mode comes back in the page (`returnFromShop`), since a load would forget
the fakes. Two seconds later, as the webhook: a flag sent is paid (`FakeFlagSubmissions.hold`, its time held), an
approved one opens (`openApproved`, a week, a month or a year), and a founded one gets its time
(`FakeFlagsBackend.addTime`). A month or a year adds a renewal, which Stop renewal stops. `approveSubmission()` opens a
paid flag at once with its time, as the server's release; `rejectSubmission()` of a paid one makes it unpaid, as the
refund. A paid flag is not replaced.

Sign in (`signIn()`), send a flag from the shop or the flag picker, and pay: the flag waits, paid. Then
`approveSubmission()`. On the console:

- `shopOff()` / `shopOn()`: no plan, as a server with the shop off, or the three again.
- `refuseCheckouts("unavailable")`: the next checkouts are refused with that refusal (`not-yours`, `not-found`,
  `unavailable`, `invalid`); `refuseCheckouts()` lets them through again.

## The legal pages

`/sale` (the terms of sale) and `/refunds` are plain pages like the terms of service (see [Pages and routes](routes.md)).
The terms of service link them in "Regional flags", the privacy policy says what Stripe and we keep in "Payments", and the
home page's footer and the About modal list them beside the other two. Their text, and the privacy policy's Payments
section, carry `<!-- DRAFT: the owner reviews this text before the merge. -->`: the owner reviews it before the merge.
