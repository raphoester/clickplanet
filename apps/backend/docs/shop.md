# The shop

**The founder of an approved flag pays for its time on Stripe's own page, and the payment opens the flag** (step 4 of
the regional flags, 2026-11-24). The shop sells three plans: a week at €4 paid once, a month at €10 and a year at €60
that renew, prices with VAT. It is **Stripe Managed Payments**: Stripe, through Link, is the merchant of record. It
charges the VAT, sends the receipts, and handles refunds and disputes. The shop keeps no name, address or card.

**Off by default.** With no Stripe key or no webhook secret (`shop.stripe`), `ListPlans` answers none, the page shows no
pay button, every call that would reach Stripe is `FailedPrecondition`, and Stripe's webhook endpoint is not served. The
operator's `OpenSubmittedFlag` stays, for flags sold by hand. Production is off until the owner's Stripe setup is done
([deploy/vps/README.md](../../../deploy/vps/README.md), "The shop").

## A module of its own

`internal/shop/`, its own schema (`shop`), its own proto package (`shop.v1`, the page's `ShopService`), one postgres
connection. **It is downstream of the flags**: it asks the flags module in the flags' words, and no module hears or asks
it (`TestTheShopSpeaksTheFlagsLanguageAloneAndNoModuleSpeaksTheShops`).

- **Before a checkout it asks** `flags.v1.InternalService/GetPayable(account, target)`: may this account buy time for
  this submission or flag now? A submission: its founder alone, approved or opened. A flag: the account whose
  submission opened it, while it is open or closed less than 30 days ago.
- **Each payment's time goes to the flags module as a command**, `flags.v1.GrantTime` (a refund's as `TakeBackTime`),
  through the shop's outbox, in the transaction that keeps the payment. A command in the flags' words, not a
  `shop.v1.TimeBought` event the flags module hears: the shop already asks the flags module, so hearing the shop would
  make a cycle, and the flags module, which planet, chat, scoring, player and alerts hear, stays upstream of all.

The flags' side is in [The flags module](flags.md#paid-time).

## The concepts

`internal/shop/internal/`, one package each:

- **`plans`**: `Plan` (week, month, year; `Renews`), `Price` (its Stripe price id from the config, the amount and the
  currency the page shows), `Catalog`, all three plans or none (off).
- **`orders`**: `AccountID`, `BuyerOf` (a signed-in account alone, as a submission's founder), `Target` (a submission's
  id or a flag's code), `Order`. `Place` refuses an order without `starts_now` (`ErrNoAgreement`). `Metadata` is what
  Stripe keeps of the order (`account_id`, `submission_id` or `flag`, `plan`, `starts_now_agreed_at`), and
  `FromMetadata` reads it back from each event: `ErrNotOurs` for what the shop never sold, since the Stripe account
  sells other things.
- **`payments`**: `Payment`, kept once per Stripe object that took the money: a week's Checkout Session (`cs_`) or an
  invoice of a month or a year (`in_`), the first and each renewal. Its PaymentIntent (`pi_`), which a refund and a
  dispute name. `Pay` answers the payment and its `GrantTime`; `TakenBack` its `TakeBackTime`, once. The grant id is
  `stripe:` and the payment's id.
- **`renewals`**: `Renewal`, a Stripe subscription as Stripe told it last (`renewing`, `stopping`, `ended`, the end of
  the period paid). `Heard` keeps the latest word by the event's time. `StoppedBy` stops it for its buyer alone.
  `NoneRenewing`: no second plan for a target while a renewal runs (one stopping charges no more).
- **`notices`**: what one Stripe event tells, in the shop's words (`Paid`, `TakenBack`, `Renewal`, `Ignored`), the
  `Reader` port, and the `Inbox` of event ids acted on (`shop.stripe_events`).
- **`stripe_gateway`**: Stripe's API with stripe-go (`github.com/stripe/stripe-go/v87`, API version `2026-10-28.endive`),
  dumb: the session, the stop, an invoice's intent, and the `Reader` of the webhook. `Off` answers `plans.ErrOff`.

## The page's calls

`shop.v1.ShopService`, behind its own session check (`shop_session_checks`): `ListPlans` reads no token, the three
others any token that names the account (`Identified`).

- **`ListPlans`**: the three prices from the config, or none while off.
- **`CreateCheckout(plan, submission_id | flag, starts_now)`** (`create_checkout_usecase`): the buyer (a guest is
  `PermissionDenied`), the plan's price (`FailedPrecondition` while off), the order (`InvalidArgument` without
  `starts_now`), the flags module's word (`NotFound`, `PermissionDenied`, `FailedPrecondition`), no renewal running
  for the target (`FailedPrecondition`), then a Checkout Session. The page only goes to its `url`.
- **`ListMyRenewals`**: the caller's renewals that renew or stop, the soonest first. None for a guest.
- **`StopRenewal(renewal_id)`** (`stop_renewal_usecase`): Stripe's `cancel_at_period_end` first, then the renewal kept
  as stopping. Another account's renewal is `NotFound`; an ended one `FailedPrecondition`. No transaction is held over
  Stripe's call: Stripe's own `customer.subscription.updated` comes after, and the latest word holds.

## The Checkout Session

`POST /v1/checkout/sessions`, Stripe's hosted page (Managed Payments works with it or with Payment Links alone: no
Elements, no embedded form):

- `managed_payments[enabled]=true`; `line_items[0]` the plan's price, quantity 1;
- `mode=payment` for the week, `mode=subscription` for the month and the year;
- `client_reference_id` the account; `metadata` the order, and the same metadata on `payment_intent_data` (a week) or
  `subscription_data` (a month or a year), so each invoice of the subscription names the order
  (`parent.subscription_details.metadata`);
- `success_url` `<returnURL>?shop=paid`, `cancel_url` `<returnURL>?shop=cancelled` (see the frontend's
  [The shop](../../frontend/docs/shop.md)).

No `automatic_tax`, `payment_method_types`, `custom_text` or `invoice_creation`: Managed Payments refuses them. The
products' tax codes are set on the products in Stripe: `txcd_10201003` (video game, streamed, not a subscription) for
the week, `txcd_10201004` (subscription) for the month and the year.

**The EU's right of withdrawal.** Stripe's Managed Payments pages say only that "Stripe applies applicable consumer
protection requirements, such as regional cooling off periods where required", and that the page's footer shows
standard terms; no page says Stripe asks the buyer to waive the 14 days for content supplied at once (checked
2026-11-24), and `custom_text` is refused. So the page asks before the redirect (`starts_now`), and the order's
`starts_now_agreed_at` goes to Stripe with the payment.

## Stripe's events

`POST /stripe/webhook` on the admin listener (`receive_notice_handler`), outside the admin gate, as the mailroom's
endpoint: what it serves says who may call it. Caddy routes `/stripe/*` there with no CORS. It is mounted only when the
shop is on.

- **The signature**: stripe-go's own check (`webhook.ConstructEventWithOptions`), the `Stripe-Signature` header with
  the endpoint's `whsec_` secret, 5 minutes of tolerance. A forged or stale one is `400`; Stripe sends nothing again
  for it. An event of another API version than stripe-go's release train is `400` too: the endpoint must be made on
  `2026-10-28.endive`.
- **Once per event**: each event's id is kept in `shop.stripe_events`, in the transaction of what it did
  (`notices.Inbox`); one sent again changes nothing. `500` when it failed: Stripe sends it again for 3 days.
- **What each event does** (`stripe_gateway/notices.go`):

| event | what the shop does |
|---|---|
| `checkout.session.completed`, `checkout.session.async_payment_succeeded` | a week paid (`mode=payment`, `payment_status=paid`): the payment kept, `GrantTime` of a week. A session paid by a bank debit is told by the second event; one not paid yet, and a month's or a year's session, are left to their later event |
| `checkout.session.async_payment_failed` | nothing: no time was granted |
| `invoice.paid` | a month or a year paid, the first invoice and each renewal (Managed Payments subscriptions are Billing's, with their invoices): its intent read from Stripe (`GET /v1/invoice_payments?invoice=…&status=paid`: since 2025-03-31 an invoice no longer names it), the payment kept, `GrantTime` of the plan's period |
| `charge.refunded` | refunded in full: the payment's `TakeBackTime`, once. In part: nothing (the operator refunds the unused time of a flag closed already) |
| `charge.dispute.closed` | `lost`: the payment's `TakeBackTime`, once. Won or else: nothing |
| `customer.subscription.created`, `.updated`, `.deleted` | the renewal as told: renewing (active, trialing, past due), stopping (`cancel_at_period_end` or `cancel_at`), ended (deleted, cancelled, never paid, unpaid, paused) |
| any other, or one the shop never sold | `200`, nothing |

**What Stripe's pages do not say** (checked 2026-11-24), so the choice is ours:

- Which events a refund and a dispute send under Managed Payments. The pages say Stripe "handles disputes for you" and
  "might issue a refund without your approval" within 60 days; Billing's page names `charge.refunded` and
  `charge.dispute.created`. The shop takes back on `charge.refunded` (in full) and on `charge.dispute.closed` (lost),
  the dispute's final word. If Managed Payments sends neither to the account, a refund keeps the flag's time: the
  operator then closes the flag by hand (`flags_close_flag`).
- Whether the renewal under Managed Payments is told as `invoice.paid`: the pages say an invoice is made each period,
  and Stripe's guide for subscriptions names `invoice.paid`. The first payment of a month or a year is its first
  invoice, so it opens the flag the same way.

## The schema

`shop.payments` (one per payment, its order, intent, amount, `payment_state`), `shop.renewals` (one per subscription,
`renewal_state`, the period's end, `decided_at`), `shop.stripe_events`, and the outbox (`outbox_event`:
`flags.v1.GrantTime`, `flags.v1.TakeBackTime`). Enums `plan`, `target_kind`. A deleted account's payments keep its id:
the accounts keep the payment (the privacy policy says so).

## Tests

`stripe_gateway`'s tests read signed events (`webhook.GenerateTestSignedPayload`: each kind, a forged one, a stale one,
another API version, the account's other sales) and send the gateway's calls to a fake Stripe (`httptest`), never the
real one: the session's form (`managed_payments[enabled]`, the mode, the metadata, the URLs), the stop, an invoice's
intent. `paid_test.go` posts signed events through the endpoint with postgres and Redis, and sees `GrantTime` and
`TakeBackTime` reach the flags' command streams once.
