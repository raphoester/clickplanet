import {Me} from "../../backends/account.ts"
import {addressOf, MarketingBackend, MarketingFailure, marketingFailureOf, SubscriptionState} from "../../backends/marketing.ts"

export type SeasonEmailsState =
    | {kind: "hidden"}
    | {kind: "guest"}
    | {kind: "ready", state: SubscriptionState, address: string, busy?: true, failure?: MarketingFailure}

type Ready = Extract<SeasonEmailsState, {kind: "ready"}>

export class SeasonEmailsStore {
    private current: SeasonEmailsState = {kind: "hidden"}
    private followed?: Me
    private generation = 0
    private readonly listeners = new Set<() => void>()

    constructor(private readonly backend: MarketingBackend) {
    }

    public state = (): SeasonEmailsState => this.current

    public subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener)
        return () => this.listeners.delete(listener)
    }

    public async follow(me: Me | undefined): Promise<void> {
        if (me === this.followed) return
        this.followed = me
        const generation = ++this.generation
        this.set({kind: "hidden"})
        if (!me) return

        try {
            if (me.linked.length === 0) {
                const offered = await this.backend.offered()
                if (generation === this.generation) this.set(offered ? {kind: "guest"} : {kind: "hidden"})
                return
            }

            const read = await this.backend.subscription()
            if (generation !== this.generation) return
            this.set(read ? {kind: "ready", state: read.state, address: read.address} : {kind: "hidden"})
        } catch (e) {
            if (generation !== this.generation) return
            console.error("Could not read the season emails", e)
            this.set({kind: "hidden"})
        }
    }

    public async refresh(): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.state !== "waiting") return
        const generation = this.generation

        let read
        try {
            read = await this.backend.subscription()
        } catch {
            return
        }

        const now = this.current
        if (generation !== this.generation || !read || now.kind !== "ready" || now.busy) return
        this.set({kind: "ready", state: read.state, address: read.address})
    }

    public async optIn(typed: string): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.state !== "none") return

        const address = addressOf(typed)
        this.set({kind: "ready", state: ready.state, address: ready.address, busy: true})
        const generation = this.generation

        try {
            const state = await this.backend.subscribe(address)
            if (generation === this.generation) this.set({kind: "ready", state, address})
        } catch (e) {
            if (generation === this.generation) this.refused(ready, marketingFailureOf(e))
        }
    }

    public async optOut(): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.state === "none") return

        this.set({...ready, busy: true, failure: undefined})
        const generation = this.generation

        try {
            await this.backend.unsubscribe()
            if (generation === this.generation) this.set({kind: "ready", state: "none", address: ready.address})
        } catch (e) {
            if (generation === this.generation) this.refused(ready, marketingFailureOf(e))
        }
    }

    private refused(ready: Ready, failure: MarketingFailure) {
        if (failure === "off") {
            this.set({kind: "hidden"})
            return
        }
        if (failure === "guest" || failure === "notSignedIn") {
            this.set({kind: "guest"})
            return
        }
        this.set({kind: "ready", state: ready.state, address: ready.address, failure})
    }

    private set(state: SeasonEmailsState) {
        this.current = state
        this.listeners.forEach((listener) => listener())
    }
}
