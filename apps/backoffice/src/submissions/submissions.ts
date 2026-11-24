import {FlagsAdmin, PlayerAdmin} from "../clients.ts"
import {learnFlags, listedOf} from "../flags.ts"
import {distanceKm, Place} from "../flags/pin.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag, RejectionReason, SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"

// From this share of a picture in colours past its 8 most common, it is most likely a photo: the game's own flags hold
// 0 to 10%, photos 24 to 65%.
export const PHOTO_FROM_PERCENT = 15

// A tile shows a flag as a disc: the middle of the picture, its height whole. About 50 CSS px wide at the closest zoom
// on a phone, 75 on a screen 1,000 px high; the tiles fill the frame from 8 px.
export const TILE_SIZES = [50, 16]

// The flags near a pin worth a look, and how many of them to show.
export const NEAR_KM = 1000
export const NEAREST = 5

// What the page read: the submissions as the server orders them, the catalog of flags, and each founder's name (unset
// when the names could not be read).
export type Listed = {submissions: KeptSubmission[], flags: Flag[], names: Map<string, string> | undefined}

export async function listOf(flags: FlagsAdmin, player: PlayerAdmin, signal: AbortSignal): Promise<Listed> {
    const [{submissions}, catalog] = await Promise.all([
        flags.listSubmissions({}, {signal}),
        flags.listFlags({}, {signal}),
    ])
    learnFlags(listedOf(catalog.flags))
    const accountIds = [...new Set(submissions.map(each => each.accountId))]
    if (accountIds.length === 0) return {submissions, flags: catalog.flags, names: new Map()}
    try {
        const answer = await player.getPlayers({accountIds}, {signal})
        return {submissions, flags: catalog.flags, names: new Map(answer.players.map(each => [each.accountId, each.name]))}
    } catch (err) {
        if (signal.aborted) throw err
        return {submissions, flags: catalog.flags, names: undefined}
    }
}

export type Filter = "waiting" | "toOpen" | "all"

export function waiting(submission: KeptSubmission): boolean {
    return submission.state === SubmissionState.WAITING
}

// Approved, a new flag: it opens once paid for.
export function toOpen(submission: KeptSubmission): boolean {
    return submission.state === SubmissionState.APPROVED && !submission.change
}

export function shownOf(submissions: KeptSubmission[], filter: Filter): KeptSubmission[] {
    switch (filter) {
        case "waiting":
            return submissions.filter(waiting)
        case "toOpen":
            return submissions.filter(toOpen)
        case "all":
            return submissions
    }
}

// After a verdict: the next waiting one below the one decided, or else the first one left.
export function nextOf(waitingBefore: KeptSubmission[], decided: string): string {
    const at = waitingBefore.findIndex(each => each.id === decided)
    const left = waitingBefore.filter(each => each.id !== decided)
    return (left.find(each => waitingBefore.indexOf(each) > at) ?? left[0])?.id ?? ""
}

export function stateLabel(state: SubmissionState): string {
    switch (state) {
        case SubmissionState.WAITING:
            return "Waiting"
        case SubmissionState.APPROVED:
            return "Approved"
        case SubmissionState.REJECTED:
            return "Rejected"
        case SubmissionState.OPENED:
            return "Opened"
        case SubmissionState.UNSPECIFIED:
            return "Unknown"
    }
}

export const REASONS: {reason: RejectionReason, label: string}[] = [
    {reason: RejectionReason.NOT_A_PLACE, label: "Not a place's flag"},
    {reason: RejectionReason.EXISTS, label: "Has a flag already"},
    {reason: RejectionReason.UNREADABLE, label: "Unreadable on a tile"},
    {reason: RejectionReason.HATEFUL, label: "Hateful"},
    {reason: RejectionReason.OTHER, label: "Other"},
]

export function reasonLabel(reason: RejectionReason): string {
    return REASONS.find(each => each.reason === reason)?.label ?? "No reason"
}

export function looksLikePhoto(submission: KeptSubmission): boolean {
    return submission.picture !== undefined && submission.otherColorsPercent >= PHOTO_FROM_PERCENT
}

// The founder's name as a list names it, or its account id's first group when the player read names none (a deleted
// account keeps its id).
export function founderOf(accountId: string, names: Map<string, string> | undefined): string {
    const name = names?.get(accountId)
    if (name !== undefined && name !== "") return name
    const [first] = accountId.split("-")
    return first.length < accountId.length ? `${first}…` : accountId
}

// What a submission is: a new flag, or a change of the flag it names.
export function kindOf(submission: KeptSubmission, flags: Flag[]): string {
    if (!submission.change) return "New flag"
    return `Change of ${flags.find(flag => flag.code === submission.flag)?.name ?? submission.flag}`
}

// Another flag, sold or the game's own, or another submission, as the open one is compared with it.
export type Other =
    | {kind: "flag", flag: Flag}
    | {kind: "submission", submission: KeptSubmission}

export type Near = {other: Other, place: Place, km: number}

// The other flags' pins and the other new flags submitted, nearest first, within NEAR_KM: the NEAREST of them. The
// flag a submission changes or opened is the submission's own, and a submission opened is its flag.
export function nearOf(submission: KeptSubmission, flags: Flag[], submissions: KeptSubmission[]): Near[] {
    const pin = submission.pin
    if (pin === undefined) return []
    const near: Near[] = []
    for (const flag of flags) {
        const at = flag.sold?.pin
        if (at === undefined || flag.code === submission.flag) continue
        near.push({other: {kind: "flag", flag}, place: at, km: distanceKm(pin, at)})
    }
    for (const other of submissions) {
        if (other.id === submission.id || other.change || other.state === SubmissionState.OPENED || other.pin === undefined) continue
        near.push({other: {kind: "submission", submission: other}, place: other.pin, km: distanceKm(pin, other.pin)})
    }
    return near.filter(each => each.km <= NEAR_KM).sort((a, b) => a.km - b.km).slice(0, NEAREST)
}

// Every other flag, the game's own included, and every other submission whose name is this one's, case and accents
// aside.
export function sameNameOf(submission: KeptSubmission, flags: Flag[], submissions: KeptSubmission[]): Other[] {
    const name = foldName(submission.name)
    if (name === "") return []
    return [
        ...flags.filter(flag => flag.code !== submission.flag && foldName(flag.name) === name)
            .map((flag): Other => ({kind: "flag", flag})),
        ...submissions.filter(other => other.id !== submission.id && other.state !== SubmissionState.OPENED
            && foldName(other.name) === name)
            .map((other): Other => ({kind: "submission", submission: other})),
    ]
}

// The letters NFD leaves whole, folded as the game's search folds them.
const LETTERS: Record<string, string> = {ß: "ss", æ: "ae", œ: "oe", ø: "o", ł: "l", đ: "d", ð: "d", þ: "th", ı: "i", ħ: "h"}

// A name as two names are compared: no case, no accents.
export function foldName(name: string): string {
    return name.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase()
        .replace(/[ßæœøłđðþıħ]/g, letter => LETTERS[letter] ?? letter).trim()
}
