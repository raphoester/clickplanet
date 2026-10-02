export type ClickBudget = {
    tokens: number

    capacity: number

    perSecond: number

    price?: ClickPrice

    linkedMultiplier?: number

    sharedWith?: SharedBy

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

export function tokensAt(budget: ClickBudget, at: number): number {
    const elapsed = Math.max(0, at - budget.readAt) / 1000
    const refilled = budget.tokens + elapsed * budget.perSecond

    return Math.max(0, Math.min(budget.capacity, refilled))
}

export function nextClickProgress(budget: ClickBudget, at: number): number {
    const tokens = tokensAt(budget, at)
    if (tokens >= budget.capacity) return 1

    return tokens - Math.floor(tokens)
}

export function secondsToNextClick(budget: ClickBudget, at: number): number {
    const tokens = tokensAt(budget, at)
    if (tokens >= 1) return 0
    if (budget.perSecond <= 0) return Infinity

    return (1 - tokens) / budget.perSecond
}

export function secondsToOneMore(budget: ClickBudget, at: number): number | undefined {
    const tokens = tokensAt(budget, at)
    if (tokens >= budget.capacity || budget.perSecond <= 0) return undefined

    return (Math.floor(tokens) + 1 - tokens) / budget.perSecond
}

export interface ClickBudgetSource {
    watchClickBudget(callback: (budget: ClickBudget) => void): () => void

    priceFor(countryId: string): void
}
