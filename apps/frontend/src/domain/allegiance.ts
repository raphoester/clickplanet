import {Countries} from "./countries.ts";
import {describeEnd} from "./endTime.ts";

export type Allegiance = {
    countryId: string

    // Milliseconds since the epoch; undefined for a flag never changed.
    changeableAt?: number

    // The rules in force let no flag change, whatever changeableAt says.
    locked?: boolean
}

// Why the flag held cannot change now: the rules in force, or the wait after a change.
export type Lock = {kind: "rules"} | {kind: "wait", until: number}

export function lockOf(allegiance: Allegiance | undefined, now: number): Lock | undefined {
    if (allegiance?.locked) return {kind: "rules"}
    if (allegiance?.changeableAt !== undefined && allegiance.changeableAt > now) return {kind: "wait", until: allegiance.changeableAt}
    return undefined
}

export function minutesLeftOf(changeableAt: number, now: number): number {
    return Math.max(1, Math.ceil((changeableAt - now) / 60_000))
}

export function heldLine(allegiance: Allegiance, now: number): string {
    const name = Countries.get(allegiance.countryId)?.name ?? allegiance.countryId.toUpperCase()
    const lock = lockOf(allegiance, now)
    if (lock?.kind !== "wait") return `You play for ${name}`
    return `You play for ${name} until ${describeEnd(lock.until, now)}`
}
