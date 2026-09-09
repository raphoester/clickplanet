import {PromiseClient} from "@connectrpc/connect"
import {SessionService} from "../gen/grpc/session/v1/session_connect.ts"
import {SessionProvider, SessionUnavailableError} from "./session.ts"

/**
 * Produces a Turnstile token. Split out of the session client so the caching
 * and minting below can be tested without a widget, a script tag or a network.
 */
export type Attester = () => Promise<string>

/** Minted a minute before the server would stop accepting the current token. */
const REFRESH_MARGIN_MS = 60_000

export type SessionClientOptions = {
    refreshMarginMs?: number
    now?: () => number
}

/**
 * Holds one session and mints another when it is about to lapse.
 *
 * Concurrent clicks share a single mint: without that, the first flurry after a
 * page load would fire one Turnstile round trip per click and spend the
 * server's mint budget immediately.
 */
export class SessionClient implements SessionProvider {
    private current?: {value: string, expiresAt: number}
    private pending?: Promise<string>

    private readonly refreshMarginMs: number
    private readonly now: () => number

    constructor(
        private readonly client: PromiseClient<typeof SessionService>,
        private readonly attest: Attester,
        options: SessionClientOptions = {},
    ) {
        this.refreshMarginMs = options.refreshMarginMs ?? REFRESH_MARGIN_MS
        this.now = options.now ?? (() => Date.now())
    }

    public async token(): Promise<string> {
        const current = this.current
        if (current && this.now() < current.expiresAt - this.refreshMarginMs) {
            return current.value
        }

        return this.mint()
    }

    public invalidate(): void {
        this.current = undefined
    }

    private mint(): Promise<string> {
        if (this.pending) return this.pending

        this.pending = this.createSession().finally(() => {
            this.pending = undefined
        })

        return this.pending
    }

    /**
     * Every failure leaves as a SessionUnavailableError. A refused mint answers
     * permission_denied, which is also what a VPN-blocked click answers — left
     * bare it would send the player to the dialog telling them to turn off a
     * VPN they may not be using.
     */
    private async createSession(): Promise<string> {
        try {
            const attestationToken = await this.attest()
            const res = await this.client.createSession({attestationToken})

            this.current = {
                value: res.token,
                expiresAt: Number(res.expiresAtUnixMs),
            }

            return res.token
        } catch (e) {
            this.current = undefined
            throw new SessionUnavailableError({cause: e})
        }
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
