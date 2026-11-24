# The shop

**The founder of an approved regional flag pays for its time on Stripe's own page** (step 4 of the regional flags,
2026-11-24): the page shows the plans, asks one consent, and goes to the page `CreateCheckout` answers. Nothing about a
card is typed in the game. The backend's side is `shop.v1.ShopService` (`proto/shop/v1/shop.proto`): `ListPlans`,
`CreateCheckout`, `ListMyRenewals`, `StopRenewal`.

- **The plans**: a week paid once, a month and a year that renew, prices with VAT, in the currency's smallest unit
  (`PlanPrice.amount`, 400 is €4). The page prices each with `Intl.NumberFormat` in its language, a whole amount with no
  cents ("€4", "€10 / month").
- **The shop off on the server** answers no plan: then no plan, no pay button, no Add time and no renewal line shows,
  and an approval reads "Approved! It opens soon." as before. A server with no shop at all (`Unimplemented`) reads the
  same.

## The code

- `domain/shop.ts`: `Plan`, `PlanPrice`, `CheckoutTarget` (a submission or a flag), `Renewal`; `payable` (an approved new
  flag), `awaitsPayment`, `flagRenewal` (the renewal on a flag or on the submission that opened it), `canAddTime` (open,
  or closed less than 30 days ago) and `flagTimeOf`, the facts a founded flag's line says.
- `backends/shop.ts`: `Shop` and `ShopRefusedError` (`not-yours`, `not-found`, `unavailable`, `invalid`, the codes
  `PermissionDenied`, `NotFound`, `FailedPrecondition`, `InvalidArgument`). `ConnectShop` (`shopBackend.ts`) sends the
  token that names the account (`identity()`, as `ListMySubmissions`), `ListPlans` none; the reads are retried, and a
  checkout is not: each call is a new page on Stripe. `FakeShop` in fake mode.
- `app/shop/shopStore.ts`: `useShop`, the plans and the renewals. `useShopFeed`, in `PlayScreen`, serves it for a
  signed-in account; the flag's place reads both each time it opens (`refresh`), never at load.
- `app/shop/PlanPicker.tsx`: the plans, then the consent and Pay.
- `app/shop/shopReturn.ts`: the way back from Stripe's page.
- `Backends.toCheckout` goes to Stripe's page: `location.assign` in production.

## The offer

**In the flag's place, the latest submission approved, new (not a change), shows the plans** under "Approved! Pick a plan
to open it.", each a `menu-tile`: its name ("1 month") and its price ("€10 / month"). A pick shows it alone, then:

- **One checkbox, unchecked**: "I want my flag to start now. I know I can't cancel once it starts." This is the EU right
  of withdrawal's waiver for digital content: the server refuses a checkout without `starts_now`, and the page sends it
  only once the box is checked.
- The links to the terms of sale and the refunds (`/sale`, `/refunds`), in a new tab.
- **Back, and Pay**, off until the box is checked. Pay calls `CreateCheckout{plan, submission_id, starts_now: true}`
  and goes to its `url`. A refusal is one line under the buttons, the place keeps the plan and the box, and Pay is on
  again; any other failure is "The payment page did not open. Try again."
- The footer's "New flag" turns secondary while a picker shows: Pay is the one thing to press.

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
  cancelled.". The flag's place is closed; nothing else opens.
- **After a payment, the submissions are read every 3 seconds for 2 minutes at most**, and no more once a read finds
  none approved and unpaid (`awaitsPayment`): the server opens the flag from Stripe's webhook a moment after the buyer
  is back. The opening is then told as any other: "Bavaria is in the game!". Time added to a flag needs no read: the
  flags stream tells the flag's new end.

## In fake mode

`FakeShop` (`backends/fakeShop.ts`) sells the three plans (400, 1000 and 6000 eur) and refuses as the server does: a
guest, no consent, the shop off, a submission not approved or unknown, a flag the account did not found, closed 30 days
ago or more, or whose renewal runs, and a target already paid and not landed yet. A checkout answers
`<origin>/play?shop=paid`; `toCheckout` in fake mode comes back in the page (`returnFromShop`), since a load would forget
the fakes. Two seconds later, as the webhook, the flag opens (`FakeFlagSubmissions.openApproved`, a week, a month or a
year) or gets its time (`FakeFlagsBackend.addTime`). A month or a year adds a renewal, which Stop renewal stops.

Sign in (`signIn()`), send a flag, `approveSubmission()`, open the flag's place: the plans show. On the console:

- `shopOff()` / `shopOn()`: no plan, as a server with the shop off, or the three again.
- `refuseCheckouts("unavailable")`: the next checkouts are refused with that refusal (`not-yours`, `not-found`,
  `unavailable`, `invalid`); `refuseCheckouts()` lets them through again.

## The legal pages

`/sale` (the terms of sale) and `/refunds` are plain pages like the terms of service (see [Pages and routes](routes.md)).
The terms of service link them in "Regional flags", the privacy policy says what Stripe and we keep in "Payments", and the
home page's footer and the About modal list them beside the other two. Their text, and the privacy policy's Payments
section, carry `<!-- DRAFT: the owner reviews this text before the merge. -->`: the owner reviews it before the merge.
