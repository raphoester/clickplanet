import {AccountBackend, AuthFailure, failureOf, Intent, Me, Provider, PROVIDERS} from "../../backends/account.ts"
import {PlayerBackend, PlayerFailure, playerFailureOf} from "../../backends/player.ts"
import {SessionProvider} from "../../backends/session.ts"

export type AccountAction = "signIn" | "link" | "signOut" | "signOutEverywhere" | "deleteAccount"

export type AccountState =
    | {kind: "loading"}
    /** Nothing to show: no provider offered and no linked account to sign out of. */
    | {kind: "hidden"}
    | {
        kind: "ready"
        /** Offered by the server, in PROVIDERS order. */
        offered: Provider[]
        me: Me
        /** The action in flight. Every button waits while one is. */
        busy?: AccountAction
        /** Why the last action failed, until the next one starts. */
        failure?: AuthFailure
        /**
         * The username, once a linked account has one and it has been read.
         * Undefined for a guest, while the read is out, and when it failed.
         */
        username?: string
        /** A username being saved. Kept apart from `busy`, but neither starts while the other runs. */
        naming?: true
        /** Why the last save of a username failed, until the next one starts. */
        nameFailure?: PlayerFailure
    }

export type AccountStoreOptions = {
    /** Leaves the page for the provider. */
    navigate: (url: string) => void
    /** Keeps the provider and the intent across that trip, so the callback page can start again with them. */
    remember: (provider: Provider, intent: Intent) => void
}

/**
 * The account section's state and every transition it makes. No DOM, no React
 * and no network of its own, like `SessionClient`, so all of it is under test.
 *
 * **Every change of account invalidates the click token.** The token names the
 * account it was minted for, and the server keeps accepting it for its hour, so
 * without this a player who signed in would paint as the guest they were.
 * Invalidating makes the next click mint again with the new cookie — or with no
 * cookie after a sign-out, which gives the browser a new guest.
 *
 * **The username is read after the account, not with it.** `GetProfile` needs a
 * click token, which may mean a mint, so the section shows as soon as the
 * account is known and the name follows. Only a linked account reads it: a
 * guest has none, and should not mint just to learn that.
 */
export class AccountStore {
    private current: AccountState = {kind: "loading"}
    private loading?: Promise<void>
    /** Bumped on every change of account, so a profile read for the old one is dropped. */
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

    /**
     * Asks which providers are offered and who the cookie belongs to. A failed
     * read of the options hides the section — login is optional, so a broken one
     * is better absent than shown. A failed read of the account shows it as a
     * guest: the buttons still work. A linked account then reads its username,
     * which this does not wait for.
     */
    public load(): Promise<void> {
        // Two callers at once share one read: StrictMode mounts twice.
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

    /** From a browser with no linked provider: a known identity moves it to that identity's account. */
    public signIn(provider: Provider): Promise<void> {
        return this.go("signIn", provider, "signIn")
    }

    /** From a linked account: adds the provider to it, and never moves the browser to another account. */
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
            // The server knows better than the list read at load: take the
            // button away rather than let it fail again.
            const offered = failure === "off" ? []
                : failure === "notOffered" ? ready.offered.filter((p) => p !== provider)
                : ready.offered
            // A link with no account left: the account was gone since the load.
            if (failure === "notSignedIn") {
                this.generation++
                this.settle(offered, {linked: []}, {failure})
                return
            }
            this.settle(offered, ready.me, {failure, username: ready.username})
        }
    }

    /**
     * Leaves for the provider's page. Throws the failure. The callback page
     * calls it directly for "Try again": the store is not loaded there, and
     * that page shows its own failure.
     */
    public async leaveFor(provider: Provider, intent: Intent): Promise<void> {
        const url = await this.backend.startSignIn(provider, intent)
        this.options.remember(provider, intent)
        this.options.navigate(url)
    }

    /**
     * Called by the callback page. Throws the failure, which that page shows
     * with its own retry; on success the account is read again, since a
     * sign-in may have landed on another account than the one the browser was on.
     */
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

    /**
     * Every way out leaves the browser with no account: the player plays on and
     * the next click mints a new guest. `notSignedIn` means it was already
     * there, so it is not reported.
     */
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

    /**
     * Saves a username for a linked account. The failure is kept apart from
     * the account's own, since it is shown under the name form and not under
     * the buttons.
     */
    public async setUsername(name: string): Promise<void> {
        const ready = this.current
        if (ready.kind !== "ready" || ready.busy || ready.naming || ready.me.linked.length === 0) return

        this.set({kind: "ready", offered: ready.offered, me: ready.me, username: ready.username, naming: true})
        const generation = this.generation

        try {
            const profile = await this.player.setName(name)
            // Read again meanwhile (a sign-in landed): that state is newer.
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

    /** A failed read leaves the name unknown: the form still shows, and saving still works. */
    private async readUsername(generation: number): Promise<void> {
        let name: string
        try {
            name = (await this.player.profile()).name
        } catch {
            return
        }

        const ready = this.current
        if (generation !== this.generation || ready.kind !== "ready" || !name) return
        // A save that landed meanwhile knows better than a read sent before it.
        if (ready.naming || ready.username !== undefined) return
        this.set({...ready, username: name})
    }

    /** The ready state with the action marked busy, or undefined when nothing may start. */
    private begin(action: AccountAction) {
        if (this.current.kind !== "ready" || this.current.busy || this.current.naming) return undefined

        const ready = this.current
        this.set({kind: "ready", offered: ready.offered, me: ready.me, username: ready.username, busy: action})
        return ready
    }

    /** A username only ever survives on a linked account. */
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
