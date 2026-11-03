import {AuthFailure} from "../../backends/account.ts"

export function pingNameOf(flagName: string | undefined): string {
    return flagName ? `Join Discord and get pinged when ${flagName} is attacked` : "Join Discord and get pinged"
}

export function pingHintOf(flagName: string | undefined): string {
    return flagName ? `Get pinged when ${flagName} is attacked` : "Get pinged for every battle"
}

// What the button says once Discord's sign-in could not start.
export function pingFailureOf(failure: AuthFailure): string {
    switch (failure) {
        case "tooManyTries":
            return "Wait a minute"
        case "off":
        case "notOffered":
            return "Not available"
        default:
            return "Try again"
    }
}
