import {createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {AuthService} from "../gen/grpc/auth/v1/auth_connect.ts"
import {SessionProvider, SessionUnavailableError} from "./session.ts"
import {Config} from "./transport.ts"

/**
 * The one client that sends cookies. The server keeps the caller's account in
 * an HttpOnly cookie on the API's host, and a cross-origin fetch only carries
 * it, or stores the one the answer sets, with `credentials: "include"`.
 *
 * Only this transport sends them. Clicks, the map and the chat stay
 * credential-free: a read that carries a cookie is a read no shared cache
 * serves, and none of them needs to know who is asking.
 *
 * Minting is a POST that must not be cached and is not on the click path's
 * critical timing, so it takes neither of the click transport's two options.
 */
export function newAuthServiceClient(config: Config): PromiseClient<typeof AuthService> {
    return createPromiseClient(AuthService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
        fetch: (input, init) => globalThis.fetch(input, {...init, credentials: "include"}),
    }))
}

/**
 * Produces a Turnstile token. Split out of the session client so the caching
 * and minting below can be tested without a widget, a script tag or a network.
 */
export type Attester = () => Promise<string>

/** Minted a minute before the server would stop accepting the current token. */
const REFRESH_MARGIN_MS = 60_000

/** A token in hand, and when the server stops accepting it. */
export type HeldSession = {
    value: string
    expiresAt: number
}

/**
 * Where a held token is kept between page loads. Injected rather than read
 * here, so the minting below stays testable without a browser — the same split
 * as the attester.
 */
export type TokenStore = {
    read(): HeldSession | undefined
    write(session: HeldSession): void
    clear(): void
}

/** For a client that keeps nothing: what it minted lasts as long as the page. */
const NO_STORE: TokenStore = {
    read: () => undefined,
    write: () => {
    },
    clear: () => {
    },
}

export type SessionClientOptions = {
    refreshMarginMs?: number
    now?: () => number
    store?: TokenStore
}

/**
 * Holds one session and mints another when it is about to lapse.
 *
 * Concurrent clicks share a single mint: without that, the first flurry after a
 * page load would fire one Turnstile round trip per click and spend the
 * server's mint budget immediately.
 *
 * **A token outlives the page it was minted on**, through the `store`. Nothing
 * mints at load — a mint is a Turnstile check, and only a click is worth one —
 * so without this a reload held nothing until its first click, and everything
 * that reads the token without minting read as a caller with no account: the
 * charges came back empty, the meter showed the scope's bucket rather than the
 * player's, the stream followed the address, and presence listed nobody.
 */
export class SessionClient implements SessionProvider {
    private current?: {value: string, expiresAt: number}
    private pending?: Promise<string>
    /** Moves on every invalidation, so a mint that started before one is not kept. */
    private generation = 0

    private readonly refreshMarginMs: number
    private readonly now: () => number
    private readonly store: TokenStore

    constructor(
        private readonly client: PromiseClient<typeof AuthService>,
        private readonly attest: Attester,
        options: SessionClientOptions = {},
    ) {
        this.refreshMarginMs = options.refreshMarginMs ?? REFRESH_MARGIN_MS
        this.now = options.now ?? (() => Date.now())
        this.store = options.store ?? NO_STORE

        // A token kept from the last page load, taken only while it is live by
        // the same rule `held` applies to one minted here.
        const kept = this.store.read()
        if (kept && this.now() < kept.expiresAt - this.refreshMarginMs) this.current = kept
    }

    public async token(): Promise<string> {
        return this.held() ?? this.mint()
    }

    /**
     * Live by the same rule `token` uses, so a token inside the refresh margin
     * is not held: `token` would not hand it out either, and an announce sent
     * with it could lapse before the server reads it.
     */
    public held(): string | undefined {
        const current = this.current
        if (current && this.now() < current.expiresAt - this.refreshMarginMs) {
            return current.value
        }

        return undefined
    }

    /**
     * Also drops a mint in flight. After a sign-in or a sign-out the cookie names
     * another account, and a token minted before that would name the old one for
     * its whole hour.
     */
    public invalidate(): void {
        this.current = undefined
        this.pending = undefined
        this.generation++
        this.store.clear()
    }

    private mint(): Promise<string> {
        if (this.pending) return this.pending

        const pending = this.createSession(this.generation).finally(() => {
            if (this.pending === pending) this.pending = undefined
        })
        this.pending = pending

        return pending
    }

    /**
     * Every failure leaves as a SessionUnavailableError. A refused mint answers
     * permission_denied, which is also what a VPN-blocked click answers — left
     * bare it would send the player to the dialog telling them to turn off a
     * VPN they may not be using.
     */
    private async createSession(generation: number): Promise<string> {
        try {
            const attestationToken = await this.attest()
            const res = await this.client.createSession({attestationToken})

            if (generation === this.generation) {
                this.current = {
                    value: res.token,
                    expiresAt: Number(res.expiresAtUnixMs),
                }
                this.store.write(this.current)
            }

            return res.token
        } catch (e) {
            // Nothing is held now, so nothing kept may be restored: a mint is
            // only attempted once `held` answered none.
            if (generation === this.generation) {
                this.current = undefined
                this.store.clear()
            }
            throw new SessionUnavailableError({cause: e})
        }
    }
}

export const SESSION_STORAGE_KEY = "clickplanet-session"

/**
 * The held token in local storage, so a reload plays on as the account it
 * already had.
 *
 * It keeps the click token and never the account: the account is the `cp_sid`
 * cookie, which is HttpOnly and stays out of reach of this page. A token lapses
 * within the hour and is bound to the address that minted it, and a page that
 * could read this could mint one of its own off that cookie — so keeping it
 * here costs nothing that was not already reachable.
 *
 * A token restored on another network is refused by the server, which is the
 * case a click already retries against a fresh mint. A read that carries it
 * (the charges, the budget) answers for nobody, exactly as it did with no token
 * at all.
 *
 * Every access is wrapped: a private window throws rather than answering.
 */
export function localTokenStore(): TokenStore {
    return {
        read(): HeldSession | undefined {
            try {
                const kept = window.localStorage.getItem(SESSION_STORAGE_KEY)
                if (!kept) return undefined

                // Whatever is in there was written by some build of this page,
                // so it is read as a claim rather than trusted as the shape.
                const held = JSON.parse(kept) as Partial<HeldSession>
                if (typeof held.value !== "string" || typeof held.expiresAt !== "number") return undefined

                return {value: held.value, expiresAt: held.expiresAt}
            } catch {
                return undefined
            }
        },
        write(session: HeldSession): void {
            try {
                window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(session))
            } catch {
                // The page plays on; the next load mints one of its own.
            }
        },
        clear(): void {
            try {
                window.localStorage.removeItem(SESSION_STORAGE_KEY)
            } catch {
                // Nothing could be kept, so there is nothing to let go of.
            }
        },
    }
}

const SCRIPT_URL = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit"

/** Generous: it also covers a player reading and ticking a checkbox. */
const ATTESTATION_TIMEOUT_MS = 30_000

type TurnstileOptions = {
    sitekey: string
    action: string
    appearance: "always" | "execute" | "interaction-only"
    callback: (token: string) => void
    "error-callback": (code?: string) => void
    "timeout-callback": () => void
}

type TurnstileApi = {
    render(container: HTMLElement, options: TurnstileOptions): string | undefined
    remove(widgetId: string): void
}

declare global {
    interface Window {
        turnstile?: TurnstileApi
    }
}

let scriptPromise: Promise<TurnstileApi> | undefined

function loadTurnstile(): Promise<TurnstileApi> {
    if (scriptPromise) return scriptPromise

    scriptPromise = new Promise<TurnstileApi>((resolve, reject) => {
        if (window.turnstile) {
            resolve(window.turnstile)
            return
        }

        const script = document.createElement("script")
        script.src = SCRIPT_URL
        script.async = true
        script.defer = true

        script.onload = () => {
            if (!window.turnstile) {
                reject(new Error("the Turnstile script loaded without defining its API"))
                return
            }
            resolve(window.turnstile)
        }

        // A blocked script never loads. Everything downstream of this treats it
        // as "no session", which is a refused click rather than a broken page.
        script.onerror = () => reject(new Error("failed to load the Turnstile script"))

        document.head.appendChild(script)
    }).catch((e) => {
        scriptPromise = undefined
        throw e
    })

    return scriptPromise
}

let host: HTMLElement | undefined

function turnstileHost(): HTMLElement {
    if (host?.isConnected) return host

    host = document.createElement("div")
    host.className = "turnstile-host"
    document.body.appendChild(host)

    return host
}

/**
 * Renders a widget, resolves with its token, and removes it again.
 *
 * A fresh widget per attestation rather than one reset between uses: Turnstile
 * tokens are redeemed exactly once, and a widget that is created and destroyed
 * has no lifecycle left to get wrong.
 */
export function turnstileAttester(sitekey: string, action: string): Attester {
    return () => new Promise<string>((resolve, reject) => {
        loadTurnstile().then((turnstile) => {
            let widgetId: string | undefined
            let settled = false

            const discard = () => {
                if (widgetId === undefined) return
                turnstile.remove(widgetId)
                widgetId = undefined
            }

            const finish = (outcome: () => void) => {
                if (settled) return
                settled = true
                clearTimeout(timer)
                discard()
                outcome()
            }

            const timer = setTimeout(
                () => finish(() => reject(new Error("Turnstile did not answer in time"))),
                ATTESTATION_TIMEOUT_MS,
            )

            widgetId = turnstile.render(turnstileHost(), {
                sitekey,
                action,
                // Invisible unless Turnstile decides this visitor has to do
                // something, which for almost everyone it does not.
                appearance: "interaction-only",
                callback: (token) => finish(() => resolve(token)),
                "error-callback": (code) => finish(() => reject(new Error(`Turnstile failed: ${code ?? "unknown"}`))),
                "timeout-callback": () => finish(() => reject(new Error("the Turnstile challenge timed out"))),
            })

            if (widgetId === undefined) {
                finish(() => reject(new Error("Turnstile refused to render a widget")))
                return
            }

            // A widget that answered synchronously inside render() was settled
            // before its id existed, so it is still here to clean up.
            if (settled) discard()
        }).catch(reject)
    })
}
