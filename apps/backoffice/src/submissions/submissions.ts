import {Capital, loadCapitals} from "../boards/capitals.ts"
import {loadPlaces, Places} from "../boards/places.ts"
import {FlagsAdmin, PlayerAdmin, SupportAdmin} from "../clients.ts"
import {countryLabel} from "../countries.ts"
import {learnFlags, listedOf} from "../flags.ts"
import {distanceKm, Place} from "../flags/pin.ts"
import {JudgementVerdict, KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Anthem, Flag, RejectionReason, SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"

// From this share of a picture in colours past its 8 most common, it is most likely a photo: the game's own flags hold
// 0 to 10%, photos 24 to 65%.
export const PHOTO_FROM_PERCENT = 15

// A tile shows a flag as a disc: the middle of the picture, its height whole. About 50 CSS px wide at the closest zoom
// on a phone, 75 on a screen 1,000 px high; the tiles fill the frame from 8 px.
export const TILE_SIZES = [50, 16]

// The capitals near a submission's worth a look, and how many of the flags' and of the countries' to show.
export const NEAR_KM = 1000
export const NEAREST = 5

// What the page read: the submissions as the server orders them, the catalog of flags, and each founder's name (unset
// when the names could not be read).
export type Listed = {submissions: KeptSubmission[], flags: Flag[], names: Map<string, string> | undefined}

// The board the world plays: where its land is, and the game's countries' capitals on it.
export type Ground = {places: Places, capitals: Capital[]}

export async function loadGround(board: string): Promise<Ground> {
    const [places, capitals] = await Promise.all([loadPlaces(board), loadCapitals(board)])
    return {places, capitals}
}

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

// The submissions whose founder replied to their letter and waits for the operator, by submission id: the server's open
// letters, read once a list read.
export async function repliedOf(support: SupportAdmin, signal: AbortSignal): Promise<ReadonlySet<string>> {
    const {letters} = await support.listLetters({openOnly: true, limit: 500}, {signal})
    return new Set(letters.map(letter => letter.submissionId))
}

export type Filter = "waiting" | "toOpen" | "replies" | "all"

export function waiting(submission: KeptSubmission): boolean {
    return submission.state === SubmissionState.WAITING
}

// Approved: it opens once paid for.
export function toOpen(submission: KeptSubmission): boolean {
    return submission.state === SubmissionState.APPROVED
}

// `replied` holds the submissions whose founder's reply to their letter waits.
export function shownOf(submissions: KeptSubmission[], filter: Filter, replied: ReadonlySet<string>): KeptSubmission[] {
    switch (filter) {
        case "waiting":
            return submissions.filter(waiting)
        case "toOpen":
            return submissions.filter(toOpen)
        case "replies":
            return submissions.filter(submission => replied.has(submission.id))
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
    {reason: RejectionReason.NOT_A_PLACE, label: "Not a place's name"},
    {reason: RejectionReason.EXISTS, label: "Has a flag already"},
    {reason: RejectionReason.UNREADABLE, label: "Not like a flag"},
    {reason: RejectionReason.HATEFUL, label: "Hateful"},
    {reason: RejectionReason.OTHER, label: "Other"},
]

export function reasonLabel(reason: RejectionReason): string {
    return REASONS.find(each => each.reason === reason)?.label ?? "No reason"
}

// What the judge, a Claude model, ruled on a submission as it waited.
export function judgementLabel(verdict: JudgementVerdict): string {
    switch (verdict) {
        case JudgementVerdict.APPROVE:
            return "Approved"
        case JudgementVerdict.REJECT:
            return "Rejected"
        case JudgementVerdict.UNSURE:
            return "Unsure, left to you"
        case JudgementVerdict.UNSPECIFIED:
            return "Not judged"
    }
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

// Another flag, sold or the game's own, another submission, or one of the game's countries, as the open one is
// compared with it.
export type Other =
    | {kind: "flag", flag: Flag}
    | {kind: "submission", submission: KeptSubmission}
    | {kind: "country", country: string}

export type Around = {other: Other, place: Place}

export type Near = Around & {km: number}

// The capitals a submission's is compared with: the sold flags' but the one it opened, which is its own; the other flags
// submitted but the ones opened, whose flag is listed; and the game's countries'.
export function aroundOf(submission: KeptSubmission, flags: Flag[], submissions: KeptSubmission[], capitals: Capital[]): Around[] {
    const around: Around[] = []
    for (const flag of flags) {
        const at = flag.sold?.pin
        if (at === undefined || flag.code === submission.flag) continue
        around.push({other: {kind: "flag", flag}, place: {latitude: at.latitude, longitude: at.longitude}})
    }
    for (const other of submissions) {
        if (other.id === submission.id || other.state === SubmissionState.OPENED || other.pin === undefined) continue
        around.push({other: {kind: "submission", submission: other}, place: {latitude: other.pin.latitude, longitude: other.pin.longitude}})
    }
    for (const {country, latitude, longitude} of capitals) around.push({other: {kind: "country", country}, place: {latitude, longitude}})
    return around
}

// Within NEAR_KM of a capital, nearest first: the NEAREST flags and submissions, and the NEAREST countries.
export function nearOf(capital: Place, around: Around[]): Near[] {
    const within = around.map(each => ({...each, km: distanceKm(capital, each.place)}))
        .filter(each => each.km <= NEAR_KM)
        .sort((a, b) => a.km - b.km)
    const countries = within.filter(each => each.other.kind === "country")
    const others = within.filter(each => each.other.kind !== "country")
    return [...others.slice(0, NEAREST), ...countries.slice(0, NEAREST)].sort((a, b) => a.km - b.km)
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

// The anthem a submission's flag takes, in the operator's words: a recording of the game's library, played for a flag
// the game ships with, named by that flag. None for none.
export function anthemWords(anthem: Anthem | undefined): string {
    if (anthem?.source.case !== "libraryCode" || anthem.source.value === "") return "none"
    return `${countryLabel(anthem.source.value)}'s, from the game's recordings`
}
