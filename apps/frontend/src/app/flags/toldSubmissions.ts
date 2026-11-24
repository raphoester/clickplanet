import {SubmissionState, Told} from "../../domain/flagSubmissions.ts"

const TOLD_KEY = "clickplanet-flag-submissions-told"

const STATES: readonly SubmissionState[] = ["waiting", "approved", "rejected", "opened"]

// None where the browser keeps nothing: a decision it cannot remember telling is not told at each load.
export function toldSubmissions(): Told | undefined {
    try {
        const stored: unknown = JSON.parse(window.localStorage.getItem(TOLD_KEY) ?? "{}")
        if (typeof stored !== "object" || stored === null) return {}
        return Object.fromEntries(Object.entries(stored)
            .filter((entry): entry is [string, SubmissionState] => STATES.includes(entry[1] as SubmissionState)))
    } catch {
        return undefined
    }
}

export function keepTold(told: Told): void {
    try {
        window.localStorage.setItem(TOLD_KEY, JSON.stringify(told))
    } catch {
        // storage unavailable: toldSubmissions() says so
    }
}
