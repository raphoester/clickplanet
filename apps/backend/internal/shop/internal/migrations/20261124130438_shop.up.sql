-- What the shop sold, as Stripe told it. Stripe holds the buyers' names, addresses and cards; the shop keeps none.
CREATE TYPE plan AS ENUM ('week', 'month', 'year');
-- What the time was bought for: the submission the first payment opens, or a flag.
CREATE TYPE target_kind AS ENUM ('submission', 'flag');
CREATE TYPE payment_state AS ENUM ('paid', 'taken_back');

-- Each payment: a week's Checkout Session (cs_), or an invoice of a month or a year (in_), the first and each renewal.
CREATE TABLE payments (
    id            text          PRIMARY KEY,
    -- The founder who paid. A deleted account's id stays: accounting keeps the payment.
    account_id    uuid          NOT NULL,
    plan          plan          NOT NULL,
    target_kind   target_kind   NOT NULL,
    -- The submission's id, or the flag's code.
    target        text          NOT NULL,
    agreed_at     timestamptz   NOT NULL,
    -- The PaymentIntent a refund or a dispute names (pi_).
    intent        text          NOT NULL UNIQUE,
    amount        bigint        NOT NULL,
    currency      text          NOT NULL,
    state         payment_state NOT NULL,
    paid_at       timestamptz   NOT NULL,
    -- When it took its state.
    decided_at    timestamptz   NOT NULL
);

CREATE TYPE renewal_state AS ENUM ('renewing', 'stopping', 'ended');

-- Each Stripe subscription of a month or a year, as Stripe told it last.
CREATE TABLE renewals (
    id             text          PRIMARY KEY,
    account_id     uuid          NOT NULL,
    plan           plan          NOT NULL,
    target_kind    target_kind   NOT NULL,
    target         text          NOT NULL,
    agreed_at      timestamptz   NOT NULL,
    state          renewal_state NOT NULL,
    period_ends_at timestamptz   NOT NULL,
    -- When Stripe told it, or the buyer stopped it: an older word changes nothing.
    decided_at     timestamptz   NOT NULL
);

CREATE INDEX renewals_account ON renewals (account_id);

-- A rejected flag's payments and renewals, refunded and cancelled.
CREATE INDEX payments_target ON payments (target_kind, target);
CREATE INDEX renewals_target ON renewals (target_kind, target);

-- Each Stripe event acted on: Stripe sends one again until it is answered, and the shop acts on it once.
CREATE TABLE stripe_events (
    id          text        PRIMARY KEY,
    received_at timestamptz NOT NULL
);

-- What the shop asks the flags module, kept in the transaction of the payment and sent by the relay.
CREATE TYPE outbox_event AS ENUM ('flags.v1.GrantTime', 'flags.v1.TakeBackTime');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
