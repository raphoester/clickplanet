import {Duration} from "@bufbuild/protobuf"
import {DEFAULT_MUTE_SECONDS, LADDER} from "./actions.ts"

export type Choice = {seconds: number, label: string}

const HOUR = 60 * 60
const DAY = 24 * HOUR

export const MUTE_CHOICES: Choice[] = [
    {seconds: 15 * 60, label: "15 minutes"},
    {seconds: DEFAULT_MUTE_SECONDS, label: "1 hour"},
    {seconds: DAY, label: "1 day"},
    {seconds: 7 * DAY, label: "1 week"},
]

export const BAN_CHOICES: Choice[] = [
    {seconds: LADDER, label: "the ladder's next step"},
    {seconds: DAY, label: "1 day"},
    {seconds: 7 * DAY, label: "1 week"},
    {seconds: 30 * DAY, label: "30 days"},
]

export function labelOf(choices: Choice[], seconds: number): string {
    return choices.find(choice => choice.seconds === seconds)?.label ?? `${seconds} seconds`
}

// A span as the operator reads it: "3h 05m", "12m 40s", "40s".
export function spanOf(duration: Duration | undefined): string {
    const total = Number(duration?.seconds ?? 0n)
    const hours = Math.floor(total / HOUR)
    const minutes = Math.floor((total % HOUR) / 60)
    const seconds = total % 60
    if (hours > 0) return `${hours}h ${String(minutes).padStart(2, "0")}m`
    if (minutes > 0) return `${minutes}m ${String(seconds).padStart(2, "0")}s`
    return `${seconds}s`
}
