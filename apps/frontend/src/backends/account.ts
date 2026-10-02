export type Provider = "google" | "discord" | "email"

export type OAuthProvider = Exclude<Provider, "email">

export const PROVIDERS: readonly Provider[] = ["google", "discord", "email"]

export const PROVIDER_NAMES: Record<Provider, string> = {
    google: "Google",
    discord: "Discord",
    email: "email",
}

export const CODE_LENGTH = 6

export type Intent = "signIn" | "link"

export type Me = {
    linked: Provider[]
}

export interface AccountBackend {
    signInOptions(): Promise<Provider[]>

    me(): Promise<Me>

    startSignIn(provider: OAuthProvider, intent: Intent): Promise<string>

    completeSignIn(code: string, state: string): Promise<void>

    startEmailSignIn(email: string, intent: Intent): Promise<void>

    completeEmailSignIn(code: string): Promise<void>

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
    | "invalidEmail"
    | "disposableEmail"
    | "tooManyCodes"
    | "wrongCode"
    | "newCode"
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
