import {AuthFailure} from "../../backends/account.ts"

export function pingNameOf(flagName: string | undefined): string {
    return flagName ? `Join Discord and get an alert when ${flagName} is under attack` : "Join Discord and get alerts"
}

export function pingHintOf(flagName: string | undefined): string {
    return flagName ? `Get a Discord alert when ${flagName} is under attack` : "Get a Discord alert when a battle starts"
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
