import {addressOf, MarketingBackend, MarketingError, Subscription, SubscriptionState} from "./marketing.ts"

export class FakeMarketingBackend implements MarketingBackend {
    private current: Subscription

    constructor(private readonly verified: readonly string[] = []) {
        this.current = {state: "none", address: verified[0] ?? ""}
    }

    public async offered(): Promise<boolean> {
        return true
    }

    public async subscription(): Promise<Subscription> {
        return {...this.current}
    }

    public async subscribe(typed: string): Promise<SubscriptionState> {
        const address = addressOf(typed)
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(address)) throw new MarketingError("invalid")

        this.current = {state: this.verified.includes(address) ? "active" : "waiting", address}
        return this.current.state
    }

    public async unsubscribe(): Promise<void> {
        this.current = {state: "none", address: this.verified[0] ?? ""}
    }

    public confirm(): void {
        if (this.current.state === "waiting") this.current = {...this.current, state: "active"}
    }
}
