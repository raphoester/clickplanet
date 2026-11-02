export type DiscordLink = {
    offered: boolean
    linked: boolean
}

export const NOT_OFFERED: DiscordLink = {offered: false, linked: false}

export interface AlertsBackend {
    discordLink(): Promise<DiscordLink>

    startDiscordLink(): Promise<string>

    completeDiscordLink(code: string, state: string): Promise<string>
}

export type AlertsFailure =
    | "off"
    | "tooManyTries"
    | "startAgain"
    | "refused"
    | "notSignedIn"
    | "failed"

export class AlertsError extends Error {
    constructor(public readonly failure: AlertsFailure, options?: {cause?: unknown}) {
        super(`alerts request refused: ${failure}`, options)
        this.name = "AlertsError"
    }
}

export function alertsFailureOf(e: unknown): AlertsFailure {
    return e instanceof AlertsError ? e.failure : "failed"
}
