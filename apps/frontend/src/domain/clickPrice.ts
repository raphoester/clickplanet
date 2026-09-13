import {ClickPrice} from "../backends/clickBudget.ts"

/** How close to the next step a free country is warned about it, as a fraction of that step's share. */
export const NEAR_NEXT_STEP = 0.8

/**
 * What the meter says about price, or nothing when there is nothing to say: a
 * country clicking at the plain rate and not close to losing it.
 */
export function describePrice(price: ClickPrice | undefined, countryName: string): {headline: string, detail: string} | undefined {
    if (!price) return undefined

    const headline = `${countryName} holds ${percent(price.share)} of the map`
    const next = price.next

    if (price.cost > 1) {
        const then = next ? ` · ${next.cost}× at ${percent(next.share)}` : ""
        return {headline, detail: `Clicks ${price.cost}× slower${then}`}
    }

    if (next && price.share >= next.share * NEAR_NEXT_STEP) {
        return {headline, detail: `Clicks ${next.cost}× slower at ${percent(next.share)}`}
    }

    return undefined
}

/** Rounded down, so a country is never told it has reached a step it has not. */
export function percent(share: number): string {
    const value = share * 100
    if (value >= 10) return `${Math.floor(value)}%`

    return `${Math.floor(value * 10) / 10}%`
}
