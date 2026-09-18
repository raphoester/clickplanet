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

    /**
     * The most that can be banked: the server's burst, in clicks. It never
     * moves with the country, a bonus or signing in.
     */
    capacity: number

    /**
     * Clicks granted back per second, at the pace the last click set: its
     * country's slowdown, signing in and a running bonus all move it.
     */
    perSecond: number

    /**
     * How much slower the selected country's players get their clicks back.
     * `perSecond` already carries the slowdown of the last click's country;
     * this is only for saying why the refill is slower. Absent from a server
     * too old to slow a refill.
     */
    price?: ClickPrice

    /**
     * How many times faster than a guest an account signed in with a provider
     * refills, into a bank of the same size. The same for every caller, so a
     * guest can be told what signing in is worth. Absent from a server that
     * grants nothing for it.
     */
    linkedMultiplier?: number

    /**
     * When `tokens` was true, on the monotonic clock. Not a server timestamp:
     * the two clocks are unrelated, and what matters is how long *this* machine
     * has watched the bucket refill since.
     */
    readAt: number
}

/** A country's players get their clicks back slower the more of the map it holds. Every click costs one. */
export type ClickPrice = {
    /** How many times slower the refill is: 1 is the plain rate, 1.5 is half as slow again. */
    slowdown: number

    /** The country's fraction of the whole map, 0 to 1. */
    share: number

    /** Where the next step starts, and its slowdown. Undefined at the top step. */
    next?: {share: number, slowdown: number}
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
 * Seconds until the count goes up by one, whatever is in hand — or undefined
 * when nothing is being granted back: a full bucket, or no refill at all.
 */
export function secondsToOneMore(budget: ClickBudget, at: number): number | undefined {
    const tokens = tokensAt(budget, at)
    if (tokens >= budget.capacity || budget.perSecond <= 0) return undefined

    return (Math.floor(tokens) + 1 - tokens) / budget.perSecond
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
     * Says which country the meter is about, since that sets the price it
     * explains. The count does not change with it: the refill slows from the
     * next click for that country on.
     */
    priceFor(countryId: string): void
}
