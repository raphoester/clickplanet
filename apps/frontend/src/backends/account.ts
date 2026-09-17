/**
 * Who the player is: sign-in with a provider, sign-out and deletion. Optional
 * from end to end — a player who never signs in plays exactly as before, on the
 * guest account the mint gives every browser.
 */

export type Provider = "google" | "discord"

/** The order the buttons are drawn in, whatever order the server answers. */
export const PROVIDERS: readonly Provider[] = ["google", "discord"]

export const PROVIDER_NAMES: Record<Provider, string> = {
    google: "Google",
    discord: "Discord",
}

/**
 * What a trip to the provider is for. `signIn` moves the browser to the
 * identity's account when the identity is known. `link` adds it to the account
 * the browser is on, and the server refuses rather than move the browser.
 */
export type Intent = "signIn" | "link"

/** The account the browser's cookie holds. `linked` is empty for a guest and for a browser with no account yet. */
export type Me = {
    linked: Provider[]
}

export interface AccountBackend {
    /** The providers this server offers. Empty while sign-in is off. */
    signInOptions(): Promise<Provider[]>

    me(): Promise<Me>

    /** The provider's URL to send the browser to. */
    startSignIn(provider: Provider, intent: Intent): Promise<string>

    completeSignIn(code: string, state: string): Promise<void>

    signOut(): Promise<void>

    signOutEverywhere(): Promise<void>

    deleteAccount(): Promise<void>
}

/**
 * Why the server said no, in the words the UI needs. One class and not one per
 * reason, unlike a refused click: every one of these is shown the same way, as
 * one line beside the button that was pressed.
 */
export type AuthFailure =
    /** Sign-in is off on this server. */
    | "off"
    /** This provider is not offered. */
    | "notOffered"
    /** The mint budget is spent: sign-in shares it with the click token. */
    | "tooManyTries"
    /** The sign-in lapsed, or this browser did not start it. */
    | "startAgain"
    /** The provider did not accept the code. */
    | "refused"
    /** The action needs an account and the browser has none. */
    | "notSignedIn"
    /** A link refused: another account already uses this identity. */
    | "linkedElsewhere"
    /** A link refused: the account already has another user of this provider. */
    | "alreadyLinked"
    /** Anything else: the network, the server. */
    | "failed"

export class AuthError extends Error {
    constructor(public readonly failure: AuthFailure, options?: {cause?: unknown}) {
        super(`auth request refused: ${failure}`, options)
        this.name = "AuthError"
    }
}

export function failureOf(e: unknown): AuthFailure {
    return e instanceof AuthError ? e.failure : "failed"
}
