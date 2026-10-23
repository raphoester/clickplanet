import {Countries} from "./countries.ts";
import {describeEnd} from "./endTime.ts";

export type Allegiance = {
    countryId: string

    // Milliseconds since the epoch; undefined for a flag never changed.
    changeableAt?: number
}

export function lockOf(allegiance: Allegiance | undefined, now: number): Allegiance | undefined {
    return allegiance?.changeableAt !== undefined && allegiance.changeableAt > now ? allegiance : undefined
}

export function minutesLeftOf(changeableAt: number, now: number): number {
    return Math.max(1, Math.ceil((changeableAt - now) / 60_000))
}

export function heldLine(allegiance: Allegiance, now: number): string {
    const name = Countries.get(allegiance.countryId)?.name ?? allegiance.countryId.toUpperCase()
    if (allegiance.changeableAt === undefined) return `You play for ${name}`
    return `You play for ${name} until ${describeEnd(allegiance.changeableAt, now)}`
}
