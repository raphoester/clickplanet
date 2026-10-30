import {AdRefillState} from "./useAdRefill.ts"

const clock = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})

export const WATCH_AD = "Watch an ad: full bank"

export function adRefillLine(state: AdRefillState): string | undefined {
    switch (state.kind) {
        case "ready":
            return "Clicks refill on their own, or an ad fills your bank now."
        case "playing":
            return "The ad is playing."
        case "refilled":
            return "Your bank is full. Thanks for watching!"
        case "waiting":
            return `Next ad refill at ${clock.format(state.until)}. Clicks refill on their own until then.`
        case "unavailable":
            return "No ad right now. Try again in a minute: clicks refill on their own."
        case "blocked":
            return "Ads are blocked, so there is no ad refill. Clicks refill on their own."
        case "off":
            return undefined
    }
}
