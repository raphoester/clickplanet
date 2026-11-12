import {Standing, stands} from "./standing.ts";

export type Allegiance = {
    countryId: string

    // Milliseconds since the epoch; undefined for a flag never changed.
    changeableAt?: number

    // The rules in force let no flag change, whatever changeableAt says.
    locked?: boolean
}

// Why the flag held cannot change now: the rules in force, or the wait after a change.
export type Lock = {kind: "rules"} | {kind: "wait", until: number}

// A flag held that does not stand locks nothing: its player picks a flag still in at once.
export function lockOf(allegiance: Allegiance | undefined, now: number, standing?: Standing): Lock | undefined {
    if (allegiance && !stands(standing, allegiance.countryId)) return undefined
    if (allegiance?.locked) return {kind: "rules"}
    if (allegiance?.changeableAt !== undefined && allegiance.changeableAt > now) return {kind: "wait", until: allegiance.changeableAt}
    return undefined
}

export function minutesLeftOf(changeableAt: number, now: number): number {
    return Math.max(1, Math.ceil((changeableAt - now) / 60_000))
}
