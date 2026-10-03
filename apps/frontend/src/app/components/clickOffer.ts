import {SharedBy} from "../../backends/clickBudget.ts"
import {factor} from "../../domain/clickPrice.ts"

export const SHARED_WITH: Record<SharedBy, string> = {
    guests: "Shared with the guests on your network",
    network: "Shared with everyone on your network",
}

export function offerText(sharedWith: SharedBy | undefined, multiplier: number): string {
    return sharedWith === "guests" ? "Sign in: your own clicks" : `Sign in: clicks ${factor(multiplier)}× faster`
}
