import {AccountBackend, AuthFailure, failureOf, Me, Provider, PROVIDERS} from "../../backends/account.ts"
import {SessionProvider} from "../../backends/session.ts"

export type AccountAction = "signIn" | "signOut" | "signOutEverywhere" | "deleteAccount"

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
    }

export type AccountStoreOptions = {
    /** Leaves the page for the provider. */
    navigate: (url: string) => void
    /** Keeps the provider across that trip, so the callback page can start again with it. */
    remember: (provider: Provider) => void
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
 */
export class AccountStore {
    private current: AccountState = {kind: "loading"}
    private loading?: Promise<void>
    private readonly listeners = new Set<() => void>()

    constructor(
        private readonly backend: AccountBackend,
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
     * guest: the buttons still work.
     */
    public load(): Promise<void> {
        // Two callers at once share one read: StrictMode mounts twice.
        if (this.loading) return this.loading

        this.loading = Promise.all([
            this.backend.signInOptions().catch(() => [] as Provider[]),
            this.backend.me().catch((): Me => ({linked: []})),
        ]).then(([offered, me]) => this.settle(offered, me))
            .finally(() => {
                this.loading = undefined
            })
        return this.loading
    }

    public async signIn(provider: Provider): Promise<void> {
        const ready = this.begin("signIn")
        if (!ready) return

        try {
            await this.leaveFor(provider)
        } catch (e) {
            const failure = failureOf(e)
            // The server knows better than the list read at load: take the
            // button away rather than let it fail again.
            const offered = failure === "off" ? []
                : failure === "notOffered" ? ready.offered.filter((p) => p !== provider)
                : ready.offered
            this.settle(offered, ready.me, failure)
        }
    }

    /**
     * Leaves for the provider's page. Throws the failure. The callback page
     * calls it directly for "Try again": the store is not loaded there, and
     * that page shows its own failure.
     */
    public async leaveFor(provider: Provider): Promise<void> {
        const url = await this.backend.startSignIn(provider)
        this.options.remember(provider)
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
                this.settle(ready.offered, ready.me, failure)
                return
            }
        }

        this.session.invalidate()
        this.settle(ready.offered, {linked: []})
    }

    /** The ready state with the action marked busy, or undefined when nothing may start. */
    private begin(action: AccountAction) {
        if (this.current.kind !== "ready" || this.current.busy) return undefined

        const ready = this.current
        this.set({kind: "ready", offered: ready.offered, me: ready.me, busy: action})
        return ready
    }

    private settle(offered: Provider[], me: Me, failure?: AuthFailure) {
        const ordered = PROVIDERS.filter((p) => offered.includes(p))
        if (ordered.length === 0 && me.linked.length === 0) {
            this.set({kind: "hidden"})
            return
        }
        this.set({kind: "ready", offered: ordered, me, failure})
    }

    private set(state: AccountState) {
        this.current = state
        this.listeners.forEach((listener) => listener())
    }
}
