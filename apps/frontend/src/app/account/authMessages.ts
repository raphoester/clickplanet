import {AuthFailure, Provider, PROVIDER_NAMES} from "../../backends/account.ts"
import {PlayerFailure} from "../../backends/player.ts"

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
        case "invalidEmail":
            return "This is not an email address."
        case "disposableEmail":
            return "Use an address you keep. Throwaway addresses are not accepted."
        case "tooManyCodes":
            return "Too many codes asked for. Wait a few minutes, then try again."
        case "wrongCode":
            return "This code is not correct. Look at the email again, then type it."
        case "newCode":
            return "This code does not work any more: it is too old, or too many codes were wrong. Ask for a new code."
        case "failed":
            return "Something went wrong. Try again."
    }
}

export function usernameMessageOf(failure: PlayerFailure): string {
    switch (failure) {
        case "invalid":
            return "This username is not allowed."
        case "taken":
            return "Another player has this username."
        case "notSignedIn":
            return "We could not check who you are. Try again."
        case "guest":
            return "Sign in to choose a username."
        case "unnamed":
        case "failed":
            return "Something went wrong. Try again."
    }
}

export function colorMessageOf(failure: PlayerFailure): string {
    switch (failure) {
        case "invalid":
            return "This color is not offered."
        case "unnamed":
            return "Pick a username first."
        case "notSignedIn":
            return "We could not check who you are. Try again."
        case "guest":
            return "Sign in to choose a color."
        case "taken":
        case "failed":
            return "Something went wrong. Try again."
    }
}

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
        case "invalidEmail":
        case "disposableEmail":
        case "tooManyCodes":
        case "wrongCode":
        case "newCode":
            return "none"
    }
}

export function providerList(providers: Provider[]): string {
    return providers.map((p) => PROVIDER_NAMES[p]).join(" and ")
}
