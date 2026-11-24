import type {CheckoutTarget, Plan, PlanPrice, Renewal} from "../domain/shop.ts"

// startsNow: the buyer asked for the flag to start at once, and knows the right to cancel ends when it starts.
export type CheckoutRequest = {plan: Plan, target: CheckoutTarget, startsNow: boolean}

// Time for a regional flag, paid on Stripe's own page.
export interface Shop {
    // The week, the month and the year; none while the shop is off.
    plans(): Promise<PlanPrice[]>
    // Stripe's page, which sends the buyer back to the game.
    checkout(request: CheckoutRequest): Promise<string>
    // None for a guest.
    myRenewals(): Promise<Renewal[]>
    // The renewal as it is now: stopping, its time paid kept.
    stopRenewal(id: string): Promise<Renewal>
}

// not-yours: a guest's, or another account's submission or flag; unavailable: the shop off, a submission not approved,
// a flag closed 30 days ago or more, or one whose renewal runs.
export type ShopRefusal = "not-yours" | "not-found" | "unavailable" | "invalid"

// One class and not one per reason: each is said the same way, as one line under the buttons.
export class ShopRefusedError extends Error {
    constructor(public readonly refusal: ShopRefusal, options?: {cause?: unknown}) {
        super(`the shop refused: ${refusal}`, options)
        this.name = "ShopRefusedError"
    }
}
