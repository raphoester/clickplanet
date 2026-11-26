import type {SoldFlag} from "./flags.ts"
import type {Submission} from "./flagSubmissions.ts"

export type Plan = "week" | "month" | "year"

// A plan on sale: its amount in the currency's smallest unit, taxes included (400 is 4.00 EUR).
export type PlanPrice = {plan: Plan, amount: number, currency: string, renews: boolean}

const DAYS: Record<Plan, number> = {week: 7, month: 365 / 12, year: 365}

const dailyOf = ({plan, amount}: PlanPrice): number => amount / DAYS[plan]

// A plan's price spread over a day, rounded up to the smallest unit: never told cheaper than it is.
export function perDayOf(price: PlanPrice): number {
    return Math.ceil(Number(dailyOf(price).toFixed(6)))
}

// What a plan saves by the day against the dearest plan, in whole percent rounded down: 42 for a month at €10 against
// a week at €4.
export function savingOf(price: PlanPrice, plans: readonly PlanPrice[]): number {
    const dearest = Math.max(...plans.map(dailyOf))
    return Math.floor(Number(((1 - dailyOf(price) / dearest) * 100).toFixed(6)))
}

// The plan cheapest by the day, when there is a choice.
export function bestValueOf(plans: readonly PlanPrice[]): Plan | undefined {
    if (plans.length < 2) return undefined
    return plans.reduce((best, price) => dailyOf(price) < dailyOf(best) ? price : best).plan
}

// The plan shown picked: the player's pick while it is on sale, else the best value.
export function pickedOf(plans: readonly PlanPrice[], chosen: Plan | undefined): PlanPrice | undefined {
    const best = bestValueOf(plans)
    return plans.find(({plan}) => plan === chosen) ?? plans.find(({plan}) => plan === best) ?? plans[0]
}

// What a payment is for: an approved submission of a new flag, which its first payment opens, or a flag: one the account
// holds, or one whose time ended, which the payment claims.
export type CheckoutTarget = {kind: "submission", id: string} | {kind: "flag", flag: string}

export type RenewalState = "renewing" | "stopping"

// A plan that renews, until periodEndsAt (milliseconds since the epoch): its next payment, or its end once stopping.
export type Renewal = {id: string, plan: Plan, target: CheckoutTarget, state: RenewalState, periodEndsAt: number}

// A new flag not paid yet: sent and waiting for its review, or approved before it was paid.
export function payable(submission: Submission): boolean {
    return (submission.state === "waiting" || submission.state === "approved") && !submission.paid
}

export function awaitsPayment(submissions: readonly Submission[]): boolean {
    return submissions.some(payable)
}

// The renewal of a flag: one on the flag, or on the submission that opened it.
export function flagRenewal(renewals: readonly Renewal[], flag: string, submissions: readonly Submission[]): Renewal | undefined {
    const openedFrom = new Set(submissions.filter((submission) => submission.flag === flag).map(({id}) => id))
    return renewals.find(({target}) => target.kind === "flag" ? target.flag === flag : openedFrom.has(target.id))
}

// A held flag's time, as its holder reads it: a renewal that runs or stops, the time more can be added to, or its
// removal.
export type FlagTime =
    | {kind: "renews", renewal: Renewal}
    | {kind: "stopping", renewal: Renewal}
    | {kind: "open", endsAt: number}
    | {kind: "closed", closedAt: number}
    | {kind: "removed", removedAt: number}
    | {kind: "none"}

export function flagTimeOf(sold: SoldFlag | undefined, closed: boolean, renewal: Renewal | undefined): FlagTime {
    if (sold === undefined) return {kind: "none"}
    if (sold.removed) return {kind: "removed", removedAt: sold.closedAt ?? 0}
    if (renewal) return renewal.state === "renewing" ? {kind: "renews", renewal} : {kind: "stopping", renewal}
    return closed ? {kind: "closed", closedAt: sold.closedAt ?? 0} : {kind: "open", endsAt: sold.endsAt}
}
