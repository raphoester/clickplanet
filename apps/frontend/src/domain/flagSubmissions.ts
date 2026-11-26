import type {FlagCatalog, FlagPicture, Pin} from "./flags.ts"

export type SubmissionState = "waiting" | "approved" | "rejected" | "opened"

export type RejectionReason = "not-a-place" | "exists" | "unreadable" | "hateful" | "other"

export type Rejection = {reason: RejectionReason, note: string}

// A flag the player submitted, as its player reads it; times in milliseconds since the epoch.
export type Submission = {
    id: string
    name: string
    pin?: Pin
    // The flag opened from it; empty until then.
    flag: string
    state: SubmissionState
    submittedAt: number
    decidedAt: number
    rejection?: Rejection
    picture?: FlagPicture
    // Its time is paid and held until its flag opens, at its approval; a paid one is not changed.
    paid: boolean
}

// The server's: flags.v1.SubmitFlagRequest and CreateSubmissionUploadRequest.
export const MAX_FLAG_NAME_LENGTH = 40
export const MAX_PICTURE_BYTES = 5 * 1024 * 1024
export const PICTURE_TYPES: readonly string[] = ["image/png", "image/jpeg", "image/webp"]
const MIN_PICTURE_WIDTH = 100

// What is sent: the server cuts the spaces at the ends too.
export function flagNameOf(typed: string): string {
    return typed.trim()
}

export function flagNameLength(typed: string): number {
    return [...flagNameOf(typed)].length
}

export function isValidFlagName(typed: string): boolean {
    const length = flagNameLength(typed)
    return length > 0 && length <= MAX_FLAG_NAME_LENGTH
}

export type PictureRefusal = "type" | "size" | "narrow" | "wide" | "small"

export function pictureFileRefusal(file: {type: string, size: number}): PictureRefusal | undefined {
    if (!PICTURE_TYPES.includes(file.type) || file.size === 0) return "type"
    if (file.size > MAX_PICTURE_BYTES) return "size"
    return undefined
}

// Between square and 2 wide by 1 high, at least 100 pixels wide: the shapes a tile and a far view can draw.
export function pictureShapeRefusal(width: number, height: number): PictureRefusal | undefined {
    if (width < MIN_PICTURE_WIDTH) return "small"
    if (width < height) return "narrow"
    if (width > 2 * height) return "wide"
    return undefined
}

// As the tile positions: y is the north pole, and longitude 0 is on -x.
export function pinAt(point: {x: number, y: number, z: number}): Pin {
    const length = Math.hypot(point.x, point.y, point.z) || 1
    const latitude = Math.asin(Math.max(-1, Math.min(1, point.y / length))) * 180 / Math.PI
    const longitude = Math.atan2(point.z, -point.x) * 180 / Math.PI
    return {latitude, longitude}
}

// The point on the unit sphere a pin is at, as pinAt reads one.
export function pointOfPin({latitude, longitude}: Pin): {x: number, y: number, z: number} {
    const phi = latitude * Math.PI / 180
    const lambda = longitude * Math.PI / 180
    return {x: -Math.cos(phi) * Math.cos(lambda), y: Math.sin(phi), z: Math.cos(phi) * Math.sin(lambda)}
}

// A flag the account holds now: its payment opened it or claimed it last. A removed one has the operator's note.
export type Held = {code: string, removalNote?: string}

// What an account reads of its flags: its submissions, the newest first, and the flags it holds.
export type MyFlags = {submissions: Submission[], held: Held[]}

export const NO_FLAGS: MyFlags = {submissions: [], held: []}

// One submission waits at a time: a new flag takes the place of one that waits unpaid. A paid one is not replaced.
export function replacedByNewOf(submissions: readonly Submission[]): Submission | undefined {
    return submissions.find((submission) => changeable(submission))
}

// Its player may change it while it waits and is not paid, never once paid or decided.
export function changeable({state, paid}: Submission): boolean {
    return state === "waiting" && !paid
}

// What its player reads: once a submission opened its flag, the flag's state now.
export type ShownState = SubmissionState | "closed"

export type Shown = Omit<Submission, "state"> & {state: ShownState}

export function shownOf(submission: Submission, catalog: FlagCatalog): Shown {
    if (submission.state !== "opened" || !catalog.closedFlag(submission.flag)) return submission
    return {...submission, state: "closed"}
}

export type Decided = Shown & {state: Exclude<ShownState, "waiting">}

function decided(shown: Shown): shown is Decided {
    return shown.state !== "waiting"
}

// The state each submission was last told in, by id.
export type Told = Readonly<Record<string, ShownState>>

// The decisions not told yet, the newest first: a submission approved, opened, then its flag closed, is told thrice. A
// flag another player claimed since is theirs: nothing of it is told.
export function untold(submissions: readonly Submission[], told: Told, catalog: FlagCatalog, held: readonly Held[]): Decided[] {
    const holds = new Set(held.map(({code}) => code))
    return submissions
        .filter(({state, flag}) => state !== "opened" || holds.has(flag))
        .map((submission) => shownOf(submission, catalog))
        .filter(decided)
        .filter(({id, state}) => told[id] !== state)
}

export function withTold(told: Told, fresh: readonly Decided[]): Told {
    const next: Record<string, ShownState> = {...told}
    for (const {id, state} of fresh) next[id] = state
    return next
}
