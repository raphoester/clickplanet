import {AccountBackend, AuthFailure, failureOf, Intent, Me, Provider, PROVIDERS} from "../../backends/account.ts"
import {PlayerBackend, PlayerFailure, playerFailureOf} from "../../backends/player.ts"
import {SessionProvider} from "../../backends/session.ts"

export type AccountAction = "signIn" | "link" | "signOut" | "signOutEverywhere" | "deleteAccount"

export type AccountState =
    | {kind: "loading"}
    | {kind: "hidden"}
    | {
        kind: "ready"
        offered: Provider[]
        me: Me
        busy?: AccountAction
        failure?: AuthFailure
        username?: string
        naming?: true
        nameFailure?: PlayerFailure
    }

export type AccountStoreOptions = {
    navigate: (url: string) => void
    remember: (provider: Provider, intent: Intent) => void
}

// Every change of account must invalidate the click token, which names the old account.
export class AccountStore {
    private current: AccountState = {kind: "loading"}
    private loading?: Promise<void>
    private generation = 0
    private readonly listeners = new Set<() => void>()

    constructor(
        private readonly backend: AccountBackend,
        private readonly player: PlayerBackend,
        private readonly session: SessionProvider,
        private readonly options: AccountStoreOptions,
    ) {
    }

    public state = (): AccountState => this.current

    public subscribe = (listener: () => void): (() => void) => {
        this.listeners.add(listener)
        return () => this.listeners.delete(listener)
    }

    public load(): Promise<void> {
        if (this.loading) return this.loading

        this.loading = Promise.all([
            this.backend.signInOptions().catch(() => [] as Provider[]),
            this.backend.me().catch((): Me => ({linked: []})),
        ]).then(([offered, me]) => {
            this.generation++
            this.settle(offered, me)
            if (me.linked.length > 0) void this.readUsername(this.generation)
        }).finally(() => {
                this.loading = undefined
            })
        return this.loading
    }

    public signIn(provider: Provider): Promise<void> {
        return this.go("signIn", provider, "signIn")
    }

    public link(provider: Provider): Promise<void> {
        return this.go("link", provider, "link")
    }

    private async go(action: AccountAction, provider: Provider, intent: Intent): Promise<void> {
        const ready = this.begin(action)
        if (!ready) return

        try {
            await this.leaveFor(provider, intent)
        } catch (e) {
            const failure = failureOf(e)
            const offered = failure === "off" ? []
                : failure === "notOffered" ? ready.offered.filter((p) => p !== provider)
                : ready.offered
            if (failure === "notSignedIn") {
                this.generation++
                this.settle(offered, {linked: []}, {failure})
                return
            }
            this.settle(offered, ready.me, {failure, username: ready.username})
        }
    }

    public async leaveFor(provider: Provider, intent: Intent): Promise<void> {
        const url = await this.backend.startSignIn(provider, intent)
        this.options.remember(provider, intent)
        this.options.navigate(url)
    }

    public async completeSignIn(code: string, state: string): Promise<void> {
        await this.backend.completeSignIn(code, state)
        this.session.invalidate()
        await this.load()
    }

    public signOut(): Promise<void> {
        return this.leave("signOut", () => this.backend.signOut())
    }

    public signOutEverywhere(): Promise<void> {
        return this.leave("signOutEverywhere", () => this.backend.signOutEverywhere())
    }

    public deleteAccount(): Promise<void> {
        return this.leave("deleteAccount", () => this.backend.deleteAccount())
    }

    private async leave(action: AccountAction, call: () => Promise<void>): Promise<void> {
        const ready = this.begin(action)
        if (!ready) return

        try {
            await call()
        } catch (e) {
            const failure = failureOf(e)
            if (failure !== "notSignedIn") {
                this.settle(ready.offered, ready.me, {failure, username: ready.username})
                return
            }
        }

        this.generation++
        this.session.invalidate()
        this.settle(ready.offered, {linked: []})
    }

    public async setUsername(name: string): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.naming || ready.me.linked.length === 0) return

        this.set({kind: "ready", offered: ready.offered, me: ready.me, username: ready.username, naming: true})
        const generation = this.generation

        try {
            const profile = await this.player.setName(name)
            if (generation !== this.generation) return
            this.set({kind: "ready", offered: ready.offered, me: ready.me, username: profile.name || undefined})
        } catch (e) {
            if (generation !== this.generation) return
            this.set({
                kind: "ready", offered: ready.offered, me: ready.me, username: ready.username,
                nameFailure: playerFailureOf(e),
            })
        }
    }

    private async readUsername(generation: number): Promise<void> {
        let name: string
        try {
            name = (await this.player.profile()).name
        } catch {
            return
        }

        const ready = this.current
        if (generation !== this.generation || ready.kind !== "ready" || !name) return
        if (ready.naming || ready.username !== undefined) return
        this.set({...ready, username: name})
    }

    private begin(action: AccountAction) {
        if (this.current.kind !== "ready" || this.current.busy || this.current.naming) return undefined

        const ready = this.current
        this.set({kind: "ready", offered: ready.offered, me: ready.me, username: ready.username, busy: action})
        return ready
    }

    private settle(offered: Provider[], me: Me, extra: {failure?: AuthFailure, username?: string} = {}) {
        const ordered = PROVIDERS.filter((p) => offered.includes(p))
        if (ordered.length === 0 && me.linked.length === 0) {
            this.set({kind: "hidden"})
            return
        }
        const username = me.linked.length > 0 ? extra.username : undefined
        this.set({kind: "ready", offered: ordered, me, failure: extra.failure, username})
    }

    private set(state: AccountState) {
        this.current = state
        this.listeners.forEach((listener) => listener())
    }
}
