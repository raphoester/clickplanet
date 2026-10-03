export type SubscriptionState = "none" | "waiting" | "active"

export type Subscription = {
    state: SubscriptionState
    address: string
}

export interface MarketingBackend {
    offered(): Promise<boolean>

    subscription(): Promise<Subscription | undefined>

    subscribe(address: string): Promise<SubscriptionState>

    unsubscribe(): Promise<void>
}

export type MarketingFailure =
    | "invalid"
    | "guest"
    | "alreadySubscribed"
    | "unavailable"
    | "tooManyTries"
    | "notSignedIn"
    | "off"
    | "failed"

export class MarketingError extends Error {
    constructor(public readonly failure: MarketingFailure, options?: {cause?: unknown}) {
        super(`season emails request refused: ${failure}`, options)
        this.name = "MarketingError"
    }
}

export function marketingFailureOf(e: unknown): MarketingFailure {
    return e instanceof MarketingError ? e.failure : "failed"
}

export function addressOf(typed: string): string {
    return typed.trim().toLowerCase()
}
