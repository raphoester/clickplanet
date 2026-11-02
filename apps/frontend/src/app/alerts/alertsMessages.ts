import {AlertsFailure} from "../../backends/alerts.ts"

export const PING_NAME = "Get pinged on Discord"

export function pingLabelOf(failure?: AlertsFailure): string {
    switch (failure) {
        case undefined:
            return "Get pinged"
        case "tooManyTries":
            return "Wait a minute"
        case "off":
            return "Not available"
        case "startAgain":
        case "refused":
        case "notSignedIn":
        case "failed":
            return "Try again"
    }
}

export function linkMessageOf(failure: AlertsFailure): string {
    switch (failure) {
        case "off":
            return "Discord pings are not available now."
        case "tooManyTries":
            return "Too many tries from your network. Wait one minute, then try again."
        case "startAgain":
            return "This link is too old, or it started in a different browser. Start again."
        case "refused":
            return "Discord did not accept it. Try again."
        case "notSignedIn":
            return "We could not check who you are. Try again."
        case "failed":
            return "Something went wrong. Try again."
    }
}
