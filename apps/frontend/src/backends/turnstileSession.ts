import {createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {AuthService} from "../gen/grpc/auth/v1/auth_connect.ts"
import {SessionProvider, SessionUnavailableError} from "./session.ts"
import {Config} from "./transport.ts"

export function newAuthServiceClient(config: Config): PromiseClient<typeof AuthService> {
    return createPromiseClient(AuthService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
        fetch: (input, init) => globalThis.fetch(input, {...init, credentials: "include"}),
    }))
}

export type Attester = () => Promise<string>

const REFRESH_MARGIN_MS = 60_000

export type HeldSession = {
    value: string
    expiresAt: number
}

export type TokenStore = {
    read(): HeldSession | undefined
    write(session: HeldSession): void
    clear(): void
}

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

export class SessionClient implements SessionProvider {
    private current?: {value: string, expiresAt: number}
    private pending?: Promise<string>
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

        const kept = this.store.read()
        if (kept && this.now() < kept.expiresAt - this.refreshMarginMs) this.current = kept
    }

    public async token(): Promise<string> {
        return this.held() ?? this.mint()
    }

    public held(): string | undefined {
        const current = this.current
        if (current && this.now() < current.expiresAt - this.refreshMarginMs) {
            return current.value
        }

        return undefined
    }

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
            if (generation === this.generation) {
                this.current = undefined
                this.store.clear()
            }
            throw new SessionUnavailableError({cause: e})
        }
    }
}

export const SESSION_STORAGE_KEY = "clickplanet-session"

export function localTokenStore(): TokenStore {
    return {
        read(): HeldSession | undefined {
            try {
                const kept = window.localStorage.getItem(SESSION_STORAGE_KEY)
                if (!kept) return undefined

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
                // storage unavailable
            }
        },
        clear(): void {
            try {
                window.localStorage.removeItem(SESSION_STORAGE_KEY)
            } catch {
                // storage unavailable
            }
        },
    }
}

const SCRIPT_URL = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit"

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
                appearance: "interaction-only",
                callback: (token) => finish(() => resolve(token)),
                "error-callback": (code) => finish(() => reject(new Error(`Turnstile failed: ${code ?? "unknown"}`))),
                "timeout-callback": () => finish(() => reject(new Error("the Turnstile challenge timed out"))),
            })

            if (widgetId === undefined) {
                finish(() => reject(new Error("Turnstile refused to render a widget")))
                return
            }

            // Turnstile may call back inside render(), before widgetId was set.
            if (settled) discard()
        }).catch(reject)
    })
}
