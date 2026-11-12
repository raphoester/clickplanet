import {AuthFailure} from "../../backends/account.ts"
import {t} from "../../i18n/page.ts"

export function pingNameOf(flagName: string | undefined): string {
    return flagName ? t("discord.pingName", {flag: flagName}) : t("discord.pingNameAnyFlag")
}

export function pingHintOf(flagName: string | undefined): string {
    return flagName ? t("discord.pingHint", {flag: flagName}) : t("discord.pingHintAnyFlag")
}

// What the button says once Discord's sign-in could not start.
export function pingFailureOf(failure: AuthFailure): string {
    switch (failure) {
        case "tooManyTries":
            return t("discord.failure.wait")
        case "off":
        case "notOffered":
            return t("discord.failure.notAvailable")
        case "startAgain":
        case "refused":
        case "notSignedIn":
        case "linkedElsewhere":
        case "alreadyLinked":
        case "invalidEmail":
        case "disposableEmail":
        case "tooManyCodes":
        case "wrongCode":
        case "newCode":
        case "failed":
            return t("common.retry")
    }
}
