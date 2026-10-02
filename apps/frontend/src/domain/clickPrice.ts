import {ClickPrice} from "../backends/clickBudget.ts"

export const NEAR_NEXT_STEP = 0.8

export function describePrice(price: ClickPrice | undefined, countryName: string): {headline: string, detail: string} | undefined {
    if (!price) return undefined

    const headline = `${countryName} holds ${percent(price.share)} of the map`
    const next = price.next

    if (price.slowdown > 1) {
        const then = next ? ` · ${factor(next.slowdown)}× at ${percent(next.share)}` : ""
        return {headline, detail: `Refills ${factor(price.slowdown)}× slower${then}`}
    }

    if (next && price.share >= next.share * NEAR_NEXT_STEP) {
        return {headline, detail: `Refills ${factor(next.slowdown)}× slower at ${percent(next.share)}`}
    }

    return undefined
}

export function factor(value: number): string {
    return `${Math.round(value * 100) / 100}`
}

export function percent(share: number): string {
    const value = share * 100
    if (value >= 10) return `${Math.floor(value)}%`

    return `${Math.floor(value * 10) / 10}%`
}
