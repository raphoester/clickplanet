import {AccountBackend, AuthFailure, failureOf, Intent, Me, OAuthProvider, Provider, PROVIDERS} from "../../backends/account.ts"
import {NameColor, PlayerBackend, PlayerFailure, playerFailureOf} from "../../backends/player.ts"
import {SessionProvider} from "../../backends/session.ts"

export type AccountAction = "signIn" | "link" | "sendCode" | "checkCode" | "signOut" | "signOutEverywhere" | "deleteAccount"

export type PendingCode = {address: string, intent: Intent}

export type AccountState =
    | {kind: "loading"}
    | {kind: "hidden"}
    | {
        kind: "ready"
        offered: Provider[]
        me: Me
        busy?: AccountAction
        failure?: AuthFailure
        code?: PendingCode
        username?: string
        naming?: true
        nameFailure?: PlayerFailure
        color?: NameColor
        coloring?: true
        colorFailure?: PlayerFailure
    }

type Ready = Extract<AccountState, {kind: "ready"}>

export type AccountStoreOptions = {
    navigate: (url: string) => void
    remember: (provider: OAuthProvider, intent: Intent) => void
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
            if (me.linked.length > 0) void this.readProfile(this.generation)
        }).finally(() => {
                this.loading = undefined
            })
        return this.loading
    }

    public signIn(provider: OAuthProvider): Promise<void> {
        return this.go("signIn", provider, "signIn")
    }

    public link(provider: OAuthProvider): Promise<void> {
        return this.go("link", provider, "link")
    }

    private async go(action: AccountAction, provider: OAuthProvider, intent: Intent): Promise<void> {
        const ready = this.begin(action)
        if (!ready) return

        try {
            await this.leaveFor(provider, intent)
        } catch (e) {
            this.refused(ready, provider, failureOf(e))
        }
    }

    public async sendCode(address: string, intent: Intent): Promise<void> {
        const ready = this.begin("sendCode")
        if (!ready) return

        try {
            await this.backend.startEmailSignIn(address, intent)
        } catch (e) {
            this.refused(ready, "email", failureOf(e))
            return
        }
        this.settle(ready.offered, ready.me, {username: ready.username, color: ready.color, code: {address, intent}})
    }

    public async checkCode(code: string): Promise<void> {
        if (this.current.kind !== "ready" || !this.current.code) return
        const ready = this.begin("checkCode")
        if (!ready) return

        try {
            await this.backend.completeEmailSignIn(code)
        } catch (e) {
            const failure = failureOf(e)
            const kept = failure === "wrongCode" || failure === "tooManyTries" || failure === "failed"
            this.refused(ready, "email", failure, kept ? ready.code : undefined)
            return
        }
        this.session.invalidate()
        await this.load()
    }

    public cancelCode(): void {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || !ready.code) return
        this.settle(ready.offered, ready.me, {username: ready.username, color: ready.color})
    }

    private refused(ready: Ready, provider: Provider, failure: AuthFailure, code?: PendingCode) {
        const offered = failure === "off" ? []
            : failure === "notOffered" ? ready.offered.filter((p) => p !== provider)
            : ready.offered
        if (failure === "notSignedIn") {
            this.generation++
            this.settle(offered, {linked: []}, {failure})
            return
        }
        this.settle(offered, ready.me, {failure, username: ready.username, color: ready.color, code})
    }

    public async leaveFor(provider: OAuthProvider, intent: Intent): Promise<void> {
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
                this.settle(ready.offered, ready.me, {failure, username: ready.username, color: ready.color})
                return
            }
        }

        this.generation++
        this.session.invalidate()
        this.settle(ready.offered, {linked: []})
    }

    public async setUsername(name: string): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.naming || ready.coloring || ready.me.linked.length === 0) return

        this.set({...kept(ready), naming: true})
        const generation = this.generation

        try {
            const profile = await this.player.setName(name)
            if (generation !== this.generation) return
            this.set({...kept(ready), username: profile.name || undefined})
        } catch (e) {
            if (generation !== this.generation) return
            this.set({...kept(ready), nameFailure: playerFailureOf(e)})
        }
    }

    public async setColor(color: NameColor): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.naming || ready.coloring || ready.username === undefined) return

        this.set({...kept(ready), coloring: true})
        const generation = this.generation

        try {
            const saved = await this.player.setColor(color)
            if (generation !== this.generation) return
            this.set({...kept(ready), color: saved})
        } catch (e) {
            if (generation !== this.generation) return
            this.set({...kept(ready), colorFailure: playerFailureOf(e)})
        }
    }

    private async readProfile(generation: number): Promise<void> {
        let name: string
        let color: NameColor
        try {
            ({name, color} = await this.player.profile())
        } catch {
            return
        }

        const ready = this.current
        if (generation !== this.generation || ready.kind !== "ready" || !name) return
        if (ready.naming || ready.coloring || ready.username !== undefined) return
        this.set({...ready, username: name, color})
    }

    private begin(action: AccountAction) {
        if (this.current.kind !== "ready" || this.current.busy || this.current.naming || this.current.coloring) return undefined

        const ready = this.current
        this.set({...kept(ready), busy: action})
        return ready
    }

    private settle(
        offered: Provider[],
        me: Me,
        extra: {failure?: AuthFailure, username?: string, color?: NameColor, code?: PendingCode} = {},
    ) {
        const ordered = PROVIDERS.filter((p) => offered.includes(p))
        if (ordered.length === 0 && me.linked.length === 0) {
            this.set({kind: "hidden"})
            return
        }
        const linked = me.linked.length > 0
        const username = linked ? extra.username : undefined
        const color = linked && username !== undefined ? extra.color : undefined
        const code = ordered.includes("email") ? extra.code : undefined
        this.set({kind: "ready", offered: ordered, me, failure: extra.failure, username, color, code})
    }

    private set(state: AccountState) {
        this.current = state
        this.listeners.forEach((listener) => listener())
    }
}

function kept(ready: Ready): Ready {
    return {kind: "ready", offered: ready.offered, me: ready.me, username: ready.username, color: ready.color, code: ready.code}
}
