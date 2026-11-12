import {ClickPrice} from "../backends/clickBudget.ts"

const NEAR_NEXT_STEP = 0.8

export type PriceStep = NonNullable<ClickPrice["next"]>

export type PriceNote =
    | {kind: "slower", slowdown: number, next?: PriceStep}
    | {kind: "nearing", next: PriceStep}

export function priceNote(price: ClickPrice | undefined): PriceNote | undefined {
    if (!price) return undefined

    const next = price.next

    if (price.slowdown > 1) return {kind: "slower", slowdown: price.slowdown, next}

    if (next && price.share >= next.share * NEAR_NEXT_STEP) return {kind: "nearing", next}

    return undefined
}

export function factor(value: number): number {
    return Math.round(value * 100) / 100
}

// Rounded down, so a step is never claimed early: whole percents from 10%, tenths under.
export function shareShown(share: number): number {
    const value = share * 100
    if (value >= 10) return Math.floor(value) / 100

    return Math.floor(value * 10) / 1000
}
