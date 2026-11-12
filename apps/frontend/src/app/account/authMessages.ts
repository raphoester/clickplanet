import {AuthFailure, Provider, PROVIDER_NAMES} from "../../backends/account.ts"
import {PlayerFailure} from "../../backends/player.ts"
import {t, translator} from "../../i18n/page.ts"
import {Key} from "../../i18n/translator.ts"

export function messageOf(failure: AuthFailure, provider?: Provider): string {
    switch (failure) {
        case "off":
            return t("auth.off")
        case "notOffered":
            return t("auth.notOffered")
        case "tooManyTries":
            return t("auth.tooManyTries")
        case "startAgain":
            return t("auth.startAgain")
        case "refused":
            return provider ? t("auth.refusedBy", {provider: providerName(provider)}) : t("auth.refused")
        case "notSignedIn":
            return t("auth.notSignedIn")
        case "linkedElsewhere":
            return provider ? t("auth.linkedElsewhereWith", {provider: providerName(provider)}) : t("auth.linkedElsewhere")
        case "alreadyLinked":
            return provider ? t("auth.alreadyLinkedWith", {provider: providerName(provider)}) : t("auth.alreadyLinked")
        case "invalidEmail":
            return t("auth.invalidEmail")
        case "disposableEmail":
            return t("auth.disposableEmail")
        case "tooManyCodes":
            return t("auth.tooManyCodes")
        case "wrongCode":
            return t("auth.wrongCode")
        case "newCode":
            return t("auth.newCode")
        case "failed":
            return t("auth.failed")
    }
}

const USERNAME_FAILURES: Record<PlayerFailure, Key> = {
    invalid: "username.invalid",
    taken: "username.taken",
    notSignedIn: "auth.notChecked",
    guest: "username.guest",
    tooSoon: "username.tooSoon",
    unnamed: "auth.failed",
    failed: "auth.failed",
}

export function usernameMessageOf(failure: PlayerFailure): string {
    return t(USERNAME_FAILURES[failure])
}

const COLOR_FAILURES: Record<PlayerFailure, Key> = {
    invalid: "account.colorNotOffered",
    unnamed: "account.colorNeedsUsername",
    notSignedIn: "auth.notChecked",
    guest: "account.colorGuest",
    taken: "auth.failed",
    tooSoon: "auth.failed",
    failed: "auth.failed",
}

export function colorMessageOf(failure: PlayerFailure): string {
    return t(COLOR_FAILURES[failure])
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

export function providerName(provider: Provider): string {
    return provider === "email" ? t("account.email") : PROVIDER_NAMES[provider]
}

export function providerList(providers: Provider[]): string {
    return translator().list(providers.map(providerName))
}
