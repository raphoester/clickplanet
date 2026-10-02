export type Provider = "google" | "discord"

export const PROVIDERS: readonly Provider[] = ["google", "discord"]

export const PROVIDER_NAMES: Record<Provider, string> = {
    google: "Google",
    discord: "Discord",
}

export type Intent = "signIn" | "link"

export type Me = {
    linked: Provider[]
}

export interface AccountBackend {
    signInOptions(): Promise<Provider[]>

    me(): Promise<Me>

    startSignIn(provider: Provider, intent: Intent): Promise<string>

    completeSignIn(code: string, state: string): Promise<void>

    signOut(): Promise<void>

    signOutEverywhere(): Promise<void>

    deleteAccount(): Promise<void>
}

export type AuthFailure =
    | "off"
    | "notOffered"
    | "tooManyTries"
    | "startAgain"
    | "refused"
    | "notSignedIn"
    | "linkedElsewhere"
    | "alreadyLinked"
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
