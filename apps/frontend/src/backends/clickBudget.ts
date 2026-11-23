export type ClickBudget = {
    tokens: number

    capacity: number

    perSecond: number

    price?: ClickPrice

    linkedMultiplier?: number

    sharedWith?: SharedBy

    // The rules in force freeze the map: no click is taken.
    frozen?: boolean

    fastStart?: boolean

    // The last click answered took less than a whole click, inside a rally's ring of its flag: what it took, and until when
    // (the page's clock) a ring of that flag is on.
    rally?: {cost: number, until: number}

    readAt: number
}

export type SharedBy = "guests" | "network"

export type ClickPrice = {
    slowdown: number

    share: number

    next?: {share: number, slowdown: number}
}

export function now(): number {
    return performance.now()
}

// A trap may leave the bank in debt, down to minus one bank: the refill pays it back first.
export function tokensAt(budget: ClickBudget, at: number): number {
    const elapsed = Math.max(0, at - budget.readAt) / 1000
    const refilled = budget.tokens + elapsed * budget.perSecond

    return Math.max(-budget.capacity, Math.min(budget.capacity, refilled))
}

export function canClick(budget: ClickBudget | undefined, at: number, cost = 1): boolean {
    return budget === undefined || tokensAt(budget, at) >= cost
}

// The bank says the clicks cost less: the last one answered was inside a ring of its flag, and a ring of it is still on.
export function rallied(budget: ClickBudget, at: number): boolean {
    return budget.rally !== undefined && at < budget.rally.until
}

// The bank still fills: what it shows moves with the clock.
export function refilling(budget: ClickBudget, at: number): boolean {
    return budget.perSecond > 0 && tokensAt(budget, at) < budget.capacity
}

export function nextClickProgress(budget: ClickBudget, at: number): number {
    const tokens = tokensAt(budget, at)
    if (tokens >= budget.capacity) return 1
    if (tokens < 0) return 0

    return tokens - Math.floor(tokens)
}

// In debt, the wait is for the first click the bank allows again.
export function secondsToOneMore(budget: ClickBudget, at: number): number | undefined {
    const tokens = tokensAt(budget, at)
    if (tokens >= budget.capacity || budget.perSecond <= 0) return undefined

    return (Math.max(Math.floor(tokens) + 1, 1) - tokens) / budget.perSecond
}

export interface ClickBudgetSource {
    watchClickBudget(callback: (budget: ClickBudget) => void): () => void

    priceFor(countryId: string): void
}
