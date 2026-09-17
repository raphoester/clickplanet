import {AuthFailure, Provider, PROVIDER_NAMES} from "../../backends/account.ts"

/** One short line per failure. Plain words: the player did nothing wrong in most of these. */
export function messageOf(failure: AuthFailure, provider?: Provider): string {
    switch (failure) {
        case "off":
            return "Sign-in is not available now."
        case "notOffered":
            return "You cannot sign in with this service now."
        case "tooManyTries":
            return "Too many tries from your network. Wait one minute, then try again."
        case "startAgain":
            return "This sign-in is too old, or it started in a different browser. Start again."
        case "refused":
            return `${provider ? PROVIDER_NAMES[provider] : "The service"} did not accept the sign-in. Try again.`
        case "notSignedIn":
            return "You are not signed in."
        case "linkedElsewhere": {
            const name = provider ? PROVIDER_NAMES[provider] : "This"
            return `This ${name} account is already used by another ClickPlanet account. `
                + "To move it here: sign in with it, delete that account, then link it here."
        }
        case "alreadyLinked":
            return `Your account already has a ${provider ? PROVIDER_NAMES[provider] : "different"} account. You can link only one of each.`
        case "failed":
            return "Something went wrong. Try again."
    }
}

/**
 * What "Try again" does on the callback page.
 *
 * - `complete`: send the same code again. Only when the server did not use it:
 *   a spent budget is refused before the code is read, and a request that
 *   failed on the way may not have arrived.
 * - `start`: the code is spent or the flow is gone, so go back to the provider.
 * - `none`: nothing the player can do from here.
 */
export type Retry = "complete" | "start" | "none"

export function retryOf(failure: AuthFailure): Retry {
    switch (failure) {
        case "tooManyTries":
        case "failed":
            return "complete"
        case "startAgain":
        case "refused":
            return "start"
        case "off":
        case "notOffered":
        case "notSignedIn":
        case "linkedElsewhere":
        case "alreadyLinked":
            return "none"
    }
}

/** "Google", "Google and Discord". */
export function providerList(providers: Provider[]): string {
    return providers.map((p) => PROVIDER_NAMES[p]).join(" and ")
}
