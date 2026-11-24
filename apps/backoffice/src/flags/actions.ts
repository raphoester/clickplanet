import {Timestamp} from "@bufbuild/protobuf"
import {FlagsAdmin} from "../clients.ts"
import {Flag} from "../gen/grpc/flags/v1/flags_pb.ts"
import {Meanings, wordsOf} from "../refusal.ts"
import {Put} from "../videos/actions.ts"

export const ACCEPTED = "image/png,image/jpeg,image/webp"
export const PICTURES_OFF = "The flags' pictures are off on the server."
export const NO_SUCH_FLAG = "No flag is kept under this code."

// The backend's rules for a sold flag (flags.v1.AdminService).
export const MAX_NAME = 40
export const MAX_BYTES = 5 * 1024 * 1024
export const MAX_DAYS = 3660
export const COMEBACK_DAYS = 30

const HOUR_MS = 3_600_000
const DAY_MS = 24 * HOUR_MS

export type Order = "end" | "name"

// A sold flag as the page shows it.
export type Sold = {flag: Flag, picture: string, endsAt: Date, closedAt: Date | undefined, played: boolean}

export function soldOf(flags: readonly Flag[]): Sold[] {
    return flags.flatMap(flag => flag.sold === undefined ? [] : [{
        flag,
        picture: flag.sold.picture?.url ?? "",
        endsAt: flag.sold.endsAt?.toDate() ?? new Date(0),
        closedAt: flag.sold.closedAt?.toDate(),
        played: flag.sold.played,
    }])
}

// The server deletes only a closed flag nobody played for, and refuses one closed under a minute ago.
export function deletable(sold: Sold): boolean {
    return sold.closedAt !== undefined && !sold.played
}

// By end: the open ones first, the soonest to end first, then the closed ones, the last closed first.
export function sortedOf(sold: readonly Sold[], order: Order): Sold[] {
    const byName = (a: Sold, b: Sold) => a.flag.name.localeCompare(b.flag.name)
    if (order === "name") return [...sold].sort(byName)
    const open = sold.filter(each => each.closedAt === undefined).sort((a, b) => a.endsAt.getTime() - b.endsAt.getTime())
    const closed = sold.filter(each => each.closedAt !== undefined)
        .sort((a, b) => (b.closedAt?.getTime() ?? 0) - (a.closedAt?.getTime() ?? 0))
    return [...open, ...closed]
}

// "12 days left", to the nearest day, so a flag opened for 30 days reads 30; "20 h left" in its last two days. A flag
// past its end is closed by the server within the minute.
export function leftOf(endsAt: Date, now: Date): string {
    const ms = endsAt.getTime() - now.getTime()
    if (ms <= 0) return "ending now"
    if (ms < HOUR_MS) return "under an hour left"
    if (ms < 2 * DAY_MS) return `${Math.floor(ms / HOUR_MS)} h left`
    return `${Math.round(ms / DAY_MS)} days left`
}

export function daysLater(from: Date, days: number): Date {
    return new Date(from.getTime() + days * DAY_MS)
}

// Until when time added opens a closed flag again.
export function comebackUntil(closedAt: Date): Date {
    return daysLater(closedAt, COMEBACK_DAYS)
}

export type Draft = {name: string, file: File | undefined, endsAt: string}

// A datetime-local, in the operator's own time; undefined for none.
export function endOf(local: string): Date | undefined {
    const date = new Date(local)
    return local === "" || Number.isNaN(date.getTime()) ? undefined : date
}

export function lengthOf(name: string): number {
    return [...name.trim()].length
}

export function nameProblem(name: string): string | undefined {
    return lengthOf(name) > MAX_NAME ? `A name of ${MAX_NAME} characters at most.` : undefined
}

// A name the server may take as typed, other than the flag's own: undefined for none.
export function renamedOf(typed: string, current: string): string | undefined {
    const name = typed.trim()
    return name === "" || name === current || nameProblem(name) !== undefined ? undefined : name
}

// What the server would refuse of a flag to open, as it is typed. A field left empty is not one: Open waits for it.
export function draftProblems(draft: Draft, now: Date): string[] {
    const problems: string[] = []
    const name = nameProblem(draft.name)
    if (name !== undefined) problems.push(name)
    if (draft.file !== undefined && !ACCEPTED.split(",").includes(draft.file.type)) problems.push("A PNG, a JPEG or a WebP.")
    if (draft.file !== undefined && draft.file.size > MAX_BYTES) problems.push("A picture of 5 MB at most.")
    const end = endOf(draft.endsAt)
    if (end !== undefined && end <= now) problems.push("An end in the future.")
    return problems
}

export function ready(draft: Draft, now: Date): boolean {
    return lengthOf(draft.name) > 0 && draft.file !== undefined && endOf(draft.endsAt) !== undefined
        && draftProblems(draft, now).length === 0
}

// A step's own words, which the refusal shows as they are.
export class Refused extends Error {}

// Three steps: a link for the picture, the picture sent there, then the flag opened with it.
export async function openFlag(
    flags: FlagsAdmin, put: Put, name: string, file: File, endsAt: Date, onSent: (bytes: number) => void,
): Promise<Flag> {
    let link
    try {
        link = await flags.createFlagUpload({contentType: file.type, sizeBytes: BigInt(file.size)})
    } catch (err) {
        throw new Refused(wordsOf(err, {failedPrecondition: PICTURES_OFF}))
    }
    try {
        await put(link.uploadUrl, file, file.type, onSent)
    } catch (err) {
        throw new Refused(`The upload failed: ${err instanceof Error ? err.message : String(err)}.`)
    }
    const {flag} = await flags.openFlag({name: name.trim(), uploadId: link.uploadId, endsAt: Timestamp.fromDate(endsAt)})
    return answered(flag)
}

export async function addTime(flags: FlagsAdmin, code: string, days: number): Promise<Flag> {
    return answered((await flags.addTime({code, days})).flag)
}

export async function closeFlag(flags: FlagsAdmin, code: string): Promise<Flag> {
    return answered((await flags.closeFlag({code})).flag)
}

export async function renameFlag(flags: FlagsAdmin, code: string, name: string): Promise<Flag> {
    return answered((await flags.renameFlag({code, name})).flag)
}

export async function deleteFlag(flags: FlagsAdmin, code: string): Promise<void> {
    await flags.deleteFlag({code})
}

export const openRefusalOf = refusalWith({failedPrecondition: PICTURES_OFF})

export const renameRefusalOf = refusalWith({notFound: NO_SUCH_FLAG})

export const deleteRefusalOf = refusalWith({
    failedPrecondition: "The server keeps it: closed under a minute ago, or opened or played for since the list was read.",
    notFound: NO_SUCH_FLAG,
})

export const addTimeRefusalOf = refusalWith({
    failedPrecondition: `Closed over ${COMEBACK_DAYS} days ago: time added no longer opens it.`,
    notFound: NO_SUCH_FLAG,
})

export const closeRefusalOf = refusalWith({failedPrecondition: "It is closed already.", notFound: NO_SUCH_FLAG})

function refusalWith(meanings: Meanings): (err: unknown) => string {
    return err => err instanceof Refused ? err.message : wordsOf(err, meanings)
}

function answered(flag: Flag | undefined): Flag {
    if (flag === undefined) throw new Refused("The server answered no flag. Reload the list.")
    return flag
}

// A count of days as typed: a whole number the server takes, or undefined.
export function daysOf(typed: string): number | undefined {
    const days = Number(typed)
    return typed.trim() !== "" && Number.isInteger(days) && days >= 1 && days <= MAX_DAYS ? days : undefined
}
