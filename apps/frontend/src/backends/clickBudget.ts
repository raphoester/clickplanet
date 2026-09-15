/**
 * How many clicks the server will still take, and when the next one arrives.
 *
 * The server owns the bucket, and only the server decides whether a click is
 * refused. But a counter that asked the server every frame would be a round
 * trip behind on every one of them, and one that ran its own bucket would
 * drift away from the server's within a few seconds.
 *
 * So neither: the server sends a *reading plus the policy*, and the client
 * replays the same arithmetic between two readings. Every answer to a click
 * re-anchors it, so the error can never accumulate — and the reading is exact
 * again the moment the player does the thing the counter is about.
 */
export type ClickBudget = {
    /**
     * Clicks in hand at `readAt`, fractional: 6.4 means six now, and the
     * seventh in 600ms at a refill of one per second.
     *
     * Already net of the clicks this client has sent and not yet heard back
     * about, which is what keeps the counter from promising a click that the
     * server has, in truth, already spent.
     */
    tokens: number

    /** The most that can be banked: the server's burst, in clicks. */
    capacity: number

    /** Clicks granted back per second. */
    perSecond: number

    /**
     * What a click costs for the country the reading is about. The three
     * numbers above are already divided by it, so nothing here multiplies;
     * this is only for saying why the meter is narrower. Absent from a server
     * too old to price clicks.
     */
    price?: ClickPrice

    /**
     * When `tokens` was true, on the monotonic clock. Not a server timestamp:
     * the two clocks are unrelated, and what matters is how long *this* machine
     * has watched the bucket refill since.
     */
    readAt: number
}

/** A click costs more tokens the more of the map its country holds. */
export type ClickPrice = {
    /** Tokens per click: 1 is the plain rate, 1.5 is half as slow again. */
    cost: number

    /** The country's fraction of the whole map, 0 to 1. */
    share: number

    /** Where the next step starts, and what it costs. Undefined at the top step. */
    next?: {share: number, cost: number}
}

/** The monotonic clock every reading is stamped against. */
export function now(): number {
    return performance.now()
}

/**
 * What the bucket holds at `at`, replaying the server's own refill.
 *
 * Clamped at both ends: never above the burst, which is where the server stops
 * banking, and never below zero, which is where a click in flight can put it.
 */
export function tokensAt(budget: ClickBudget, at: number): number {
    const elapsed = Math.max(0, at - budget.readAt) / 1000
    const refilled = budget.tokens + elapsed * budget.perSecond

    return Math.max(0, Math.min(budget.capacity, refilled))
}

/**
 * How much of the next click has arrived, from 0 to 1 — the fraction that fills
 * the pip on screen.
 *
 * A full bucket reports 1 rather than the fraction of a token it is not
 * accumulating: nothing is pending, so nothing should look half-done.
 */
export function nextClickProgress(budget: ClickBudget, at: number): number {
    const tokens = tokensAt(budget, at)
    if (tokens >= budget.capacity) return 1

    return tokens - Math.floor(tokens)
}

/** Seconds until one more click is in hand, or 0 when one already is. */
export function secondsToNextClick(budget: ClickBudget, at: number): number {
    const tokens = tokensAt(budget, at)
    if (tokens >= 1) return 0
    if (budget.perSecond <= 0) return Infinity

    return (1 - tokens) / budget.perSecond
}

/**
 * The source of those readings: the click transport, which learns the budget
 * from the answers to its own clicks.
 */
export interface ClickBudgetSource {
    /**
     * Reports every reading, and the drop the moment a click is sent. Returns
     * the unsubscribe.
     *
     * A backend whose server does not throttle clicks never calls back, and
     * nothing is shown.
     */
    watchClickBudget(callback: (budget: ClickBudget) => void): () => void

    /**
     * Says which country the meter is about, since that sets the price. The
     * reading changes with it: a click for a country holding most of the map
     * costs more than one for a country holding none.
     */
    priceFor(countryId: string): void
}
