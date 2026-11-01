import {Duration} from "@bufbuild/protobuf"
import {clockOf} from "../components/time.ts"

// The daily cutoff, in milliseconds after midnight UTC: a day ends at it, and a battle starts or ends at it.
export type Cutoff = number

// Undefined from a server that sends none.
export function cutoffOf(duration: Duration | undefined): Cutoff | undefined {
    if (duration === undefined) return undefined
    return Number(duration.seconds) * 1000 + Math.round(duration.nanos / 1e6)
}

// The cutoff on the day of a time, in UTC.
export function cutoffOn(at: Date, cutoff: Cutoff): Date {
    return new Date(Date.UTC(at.getUTCFullYear(), at.getUTCMonth(), at.getUTCDate()) + cutoff)
}

// "Days end at 23:00.", in the operator's time on the day of a time.
export function daysEndWords(at: Date, cutoff: Cutoff): string {
    return `Days end at ${clockOf(cutoffOn(at, cutoff))}.`
}

// Why the server will refuse a battle that neither starts nor ends at the cutoff, or undefined when it does.
export function offCutoffWords(battle: string, startsAt: Date, endsAt: Date, cutoff: Cutoff): string | undefined {
    if (cutoffOn(startsAt, cutoff).getTime() === startsAt.getTime() || cutoffOn(endsAt, cutoff).getTime() === endsAt.getTime()) {
        return undefined
    }
    return `${battle} runs ${clockOf(startsAt)} to ${clockOf(endsAt)}: it must start or end at ${clockOf(cutoffOn(endsAt, cutoff))}, when a day ends.`
}
