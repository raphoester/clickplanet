import {
    AccountBackend,
    AuthFailure,
    ButtonProvider,
    failureOf,
    Intent,
    Me,
    OAuthProvider,
    Provider,
    PROVIDERS,
    SignInOutcome,
} from "../../backends/account.ts"
import {NameColor, PlayerBackend, PlayerFailure, playerFailureOf, RenameTooSoonError} from "../../backends/player.ts"
import {Fronts, PlayerTitle, ProgressionBackend, TitleDashboard} from "../../backends/progression.ts"
import {SessionProvider} from "../../backends/session.ts"
import {useBadges} from "../badges/badgesStore.ts"
import {CrazyGamesUser} from "../../backends/crazyGames.ts"
import {crazyGamesUserOf, leaving, LeaverStore} from "../../domain/crazyGames.ts"

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
        accountId?: string
        username?: string
        renamableAt?: number
        naming?: true
        nameFailure?: PlayerFailure
        color?: NameColor
        coloring?: true
        colorFailure?: PlayerFailure
        // This browser's sign-in gave the guest it was on its first identity, which earns a refill.
        guestLinked?: true
        // This browser's sign-in brought back an account its player had asked to delete, until the player is told.
        restored?: true
        // This browser's sign-in was Discord's, which joins the game's Discord server a moment after it comes back.
        joiningDiscord?: true
    }

type Ready = Extract<AccountState, {kind: "ready"}>

export type AccountStoreOptions = {
    navigate: (url: string) => void
    remember: (provider: OAuthProvider, intent: Intent) => void
    // Pledges the page's flag to an account that holds none: Discord's sign-in joins the server with the flag's role.
    claimFlag: () => Promise<void>
    // On CrazyGames its account is the only way in, and the page plays as whoever CrazyGames says is signed in.
    crazyGames?: CrazyGamesUser
    // On CrazyGames, who asked on this device to delete their account: see leaving.
    leavers?: LeaverStore
}

// Every change of account must invalidate the click token, which names the old account.
export class AccountStore {
    private current: AccountState = {kind: "loading"}
    private loading?: Promise<void>
    private following?: Promise<void>
    private generation = 0
    private readonly listeners = new Set<() => void>()

    constructor(
        private readonly backend: AccountBackend,
        private readonly player: PlayerBackend,
        private readonly progression: ProgressionBackend,
        private readonly session: SessionProvider,
        private readonly options: AccountStoreOptions,
    ) {
    }

    public state = (): AccountState => this.current

    public titles = async (): Promise<TitleDashboard> => {
        const generation = this.generation
        const dashboard = await this.progression.titles()
        this.wore(generation, dashboard.worn)
        return dashboard
    }

    public wearTitle = async (id: string): Promise<PlayerTitle | undefined> => {
        const generation = this.generation
        const worn = await this.progression.wearTitle(id)
        this.wore(generation, worn)
        return worn
    }

    public fronts = (): Promise<Fronts> => this.progression.fronts()

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

    public signIn(provider: ButtonProvider): Promise<void> {
        return provider === "crazygames" ? this.promptCrazyGames() : this.go("signIn", provider, "signIn")
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
        this.settle(ready.offered, ready.me, {...named(ready), code: {address, intent}})
    }

    public async checkCode(code: string): Promise<void> {
        if (this.current.kind !== "ready" || !this.current.code) return
        const ready = this.begin("checkCode")
        if (!ready) return

        let outcome: SignInOutcome
        try {
            outcome = await this.backend.completeEmailSignIn(code)
        } catch (e) {
            const failure = failureOf(e)
            const kept = failure === "wrongCode" || failure === "tooManyTries" || failure === "failed"
            this.refused(ready, "email", failure, kept ? ready.code : undefined)
            return
        }
        await this.signedIn(outcome)
    }

    public cancelCode(): void {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || !ready.code) return
        this.settle(ready.offered, ready.me, named(ready))
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
        this.settle(offered, ready.me, {...named(ready), failure, code})
    }

    // Discord's sign-in joins the game's server too. An account signed in already links it, so the player stays on it;
    // a guest signs in, and moves to the account that Discord user has, if any. Refused, it throws, and changes nothing.
    public connectDiscord(): Promise<void> {
        const ready = this.current
        const intent: Intent = ready.kind === "ready" && ready.me.linked.length > 0 ? "link" : "signIn"
        return this.leaveFor("discord", intent)
    }

    public async leaveFor(provider: OAuthProvider, intent: Intent): Promise<void> {
        if (provider === "discord") await this.claimFlag()
        const url = await this.backend.startSignIn(provider, intent)
        this.options.remember(provider, intent)
        this.options.navigate(url)
    }

    // A failed pledge still joins: the first click pledges, and the role follows the pledge.
    private async claimFlag(): Promise<void> {
        try {
            await this.options.claimFlag()
        } catch (e) {
            console.error("Could not pledge the flag before Discord's sign-in", e)
        }
    }

    // provider: the one the browser left for, when it is remembered.
    public async completeSignIn(code: string, state: string, provider?: OAuthProvider): Promise<void> {
        await this.signedIn(await this.backend.completeSignIn(code, state), provider)
    }

    private async signedIn(outcome: SignInOutcome, provider?: OAuthProvider): Promise<void> {
        this.session.invalidate()
        await this.load()

        const ready = this.current
        if (ready.kind !== "ready") return
        const marked: Ready = {...ready}
        if (outcome === "linked" && ready.me.linked.length === 1) marked.guestLinked = true
        if (outcome === "restored") marked.restored = true
        if (provider === "discord") marked.joiningDiscord = true
        if (marked.guestLinked || marked.restored || marked.joiningDiscord) this.set(marked)
    }

    public welcomedBack(): void {
        const ready = this.current
        if (ready.kind !== "ready" || !ready.restored) return
        this.set({...ready, restored: undefined})
    }

    // At every start, and at every sign-in CrazyGames tells: a shared device, or a player signed out there, changes who plays.
    public followCrazyGames(): Promise<void> {
        const crazyGames = this.options.crazyGames
        if (!crazyGames) return Promise.resolve()
        this.following ??= this.signInAs(crazyGames).finally(() => {
            this.following = undefined
        })
        return this.following
    }

    private async signInAs(crazyGames: CrazyGamesUser): Promise<void> {
        const token = await crazyGames.token()
        if (this.current.kind !== "ready") await this.load()
        const ready = this.current
        if (ready.kind !== "ready") return

        if (token === undefined) {
            if (ready.me.linked.includes("crazygames")) await this.signOut()
            return
        }
        if (leaving(this.options.leavers?.read(), crazyGamesUserOf(token), Date.now())) return
        if (!ready.offered.includes("crazygames") || !this.begin("signIn")) return

        let outcome: SignInOutcome
        try {
            outcome = await this.backend.completeCrazyGamesSignIn(token)
        } catch (e) {
            this.refused(ready, "crazygames", failureOf(e))
            return
        }
        await this.signedIn(outcome)
    }

    private async promptCrazyGames(): Promise<void> {
        const crazyGames = this.options.crazyGames
        if (!crazyGames) return
        const ready = this.begin("signIn")
        if (!ready) return

        const signedIn = await crazyGames.prompt()
        this.settle(ready.offered, ready.me, named(ready))
        if (!signedIn) return
        // The player asked to sign in: that brings back an account they asked to delete.
        this.options.leavers?.write(undefined)
        await this.followCrazyGames()
    }

    public async signOut(): Promise<void> {
        await this.leave("signOut", () => this.backend.signOut())
    }

    public async signOutEverywhere(): Promise<void> {
        await this.leave("signOutEverywhere", () => this.backend.signOutEverywhere())
    }

    // The server keeps the account 30 days, and a sign-in to it before then brings it back.
    public async deleteAccount(): Promise<void> {
        if (await this.leave("deleteAccount", () => this.backend.deleteAccount())) await this.rememberLeaver()
    }

    private async rememberLeaver(): Promise<void> {
        const crazyGames = this.options.crazyGames
        if (!crazyGames || !this.options.leavers) return
        const token = await crazyGames.token()
        const user = token === undefined ? undefined : crazyGamesUserOf(token)
        if (user !== undefined) this.options.leavers.write({user, at: Date.now()})
    }

    // Whether the browser left the account: false when the server refused.
    private async leave(action: AccountAction, call: () => Promise<void>): Promise<boolean> {
        const ready = this.begin(action)
        if (!ready) return false

        try {
            await call()
        } catch (e) {
            const failure = failureOf(e)
            if (failure !== "notSignedIn") {
                this.settle(ready.offered, ready.me, {...named(ready), failure})
                return false
            }
        }

        this.generation++
        this.session.invalidate()
        this.settle(ready.offered, {linked: []})
        return true
    }

    public async setUsername(name: string): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.naming || ready.coloring || ready.me.linked.length === 0) return

        this.set({...kept(ready), naming: true})
        const generation = this.generation

        try {
            const profile = await this.player.setName(name)
            if (generation !== this.generation) return
            this.set({
                ...kept(ready), accountId: profile.accountId || undefined, username: profile.name || undefined,
                renamableAt: profile.renamableAt,
            })
        } catch (e) {
            if (generation !== this.generation) return
            const renamableAt = e instanceof RenameTooSoonError ? e.renamableAt : ready.renamableAt
            this.set({...kept(ready), nameFailure: playerFailureOf(e), renamableAt})
        }
    }

    public async setColor(color: NameColor): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.naming || ready.coloring || ready.username === undefined) return

        this.set({...kept(ready), color, coloring: true})
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
        let accountId: string | undefined
        let name: string
        let color: NameColor
        let renamableAt: number | undefined
        try {
            const profile = await this.player.profile()
            accountId = profile.accountId || undefined
            name = profile.name
            color = profile.color
            renamableAt = profile.renamableAt
        } catch {
            return
        }

        const ready = this.current
        if (generation !== this.generation || ready.kind !== "ready") return
        if (!name || ready.naming || ready.coloring || ready.username !== undefined) {
            this.set({...ready, accountId})
            return
        }
        this.set({...ready, accountId, username: name, color, renamableAt})
    }

    // The title this account wears now goes beside its name everywhere on the page, before any read of its badge.
    private wore(generation: number, worn: PlayerTitle | undefined) {
        const ready = this.current
        if (generation !== this.generation || ready.kind !== "ready" || ready.accountId === undefined) return
        useBadges.getState().wear(ready.accountId, worn)
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
        extra: {
            failure?: AuthFailure, accountId?: string, username?: string, renamableAt?: number, color?: NameColor, code?: PendingCode
        } = {},
    ) {
        const ordered = PROVIDERS.filter((p) => offered.includes(p) && (p === "crazygames") === (this.options.crazyGames !== undefined))
        if (ordered.length === 0 && me.linked.length === 0) {
            this.set({kind: "hidden"})
            return
        }
        const linked = me.linked.length > 0
        const accountId = linked ? extra.accountId : undefined
        const username = linked ? extra.username : undefined
        const color = linked && username !== undefined ? extra.color : undefined
        const renamableAt = linked && username !== undefined ? extra.renamableAt : undefined
        const code = ordered.includes("email") ? extra.code : undefined
        this.set({kind: "ready", offered: ordered, me, failure: extra.failure, accountId, username, renamableAt, color, code})
    }

    private set(state: AccountState) {
        this.current = state
        this.listeners.forEach((listener) => listener())
    }
}

function kept(ready: Ready): Ready {
    return {kind: "ready", offered: ready.offered, me: ready.me, ...named(ready), code: ready.code}
}

function named(ready: Ready) {
    return {accountId: ready.accountId, username: ready.username, renamableAt: ready.renamableAt, color: ready.color}
}
