import type {SoldFlag} from "./flags.ts"
import type {Submission} from "./flagSubmissions.ts"

export type Plan = "week" | "month" | "year"

// A plan on sale: its amount in the currency's smallest unit, taxes included (400 is 4.00 EUR).
export type PlanPrice = {plan: Plan, amount: number, currency: string, renews: boolean}

// What a payment is for: an approved submission of a new flag, which its first payment opens, or a flag founded.
export type CheckoutTarget = {kind: "submission", id: string} | {kind: "flag", flag: string}

export type RenewalState = "renewing" | "stopping"

// A plan that renews, until periodEndsAt (milliseconds since the epoch): its next payment, or its end once stopping.
export type Renewal = {id: string, plan: Plan, target: CheckoutTarget, state: RenewalState, periodEndsAt: number}

// Time added within this long of a close opens the flag again.
export const COMEBACK_MS = 30 * 24 * 60 * 60 * 1000

// A flag approved, which waits for its first payment.
export function payable(submission: Submission): boolean {
    return submission.state === "approved"
}

export function awaitsPayment(submissions: readonly Submission[]): boolean {
    return submissions.some(payable)
}

// The renewal of a flag: one on the flag, or on the submission that opened it.
export function flagRenewal(renewals: readonly Renewal[], flag: string, submissions: readonly Submission[]): Renewal | undefined {
    const openedFrom = new Set(submissions.filter((submission) => submission.flag === flag).map(({id}) => id))
    return renewals.find(({target}) => target.kind === "flag" ? target.flag === flag : openedFrom.has(target.id))
}

// Open, or closed less than 30 days ago.
export function canAddTime(sold: SoldFlag | undefined, closed: boolean, now: number): boolean {
    if (sold === undefined) return false
    if (!closed) return true
    return sold.closedAt !== undefined && now - sold.closedAt < COMEBACK_MS
}

// A founded flag's time, as its founder reads it: a renewal that runs or stops, or the time more can be added to.
export type FlagTime =
    | {kind: "renews", renewal: Renewal}
    | {kind: "stopping", renewal: Renewal}
    | {kind: "open", endsAt: number}
    | {kind: "closed", closedAt: number}
    | {kind: "none"}

export function flagTimeOf(sold: SoldFlag | undefined, closed: boolean, renewal: Renewal | undefined, now: number): FlagTime {
    if (renewal) return renewal.state === "renewing" ? {kind: "renews", renewal} : {kind: "stopping", renewal}
    if (sold === undefined || !canAddTime(sold, closed, now)) return {kind: "none"}
    return closed ? {kind: "closed", closedAt: sold.closedAt ?? now} : {kind: "open", endsAt: sold.endsAt}
}
