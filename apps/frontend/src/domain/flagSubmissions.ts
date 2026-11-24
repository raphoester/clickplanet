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
    // An opened one's, while the operator's close of its flag holds: why, in the operator's words.
    closeNote?: string
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

// The newest first, as the server lists them.
export function latestOf(submissions: readonly Submission[]): Submission | undefined {
    return submissions[0]
}

// The one that waits: one at a time, and the only one its player may change.
export function waitingOf(submissions: readonly Submission[]): Submission | undefined {
    return submissions.find((submission) => submission.state === "waiting")
}

// The flags the account founded, the newest first: each opened from one of its submissions.
export function foundedFlags(submissions: readonly Submission[]): string[] {
    return submissions.filter(({state, flag}) => state === "opened" && flag !== "").map(({flag}) => flag)
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

// The decisions not told yet, the newest first: a submission approved, opened, then its flag closed, is told thrice.
export function untold(submissions: readonly Submission[], told: Told, catalog: FlagCatalog): Decided[] {
    return submissions
        .map((submission) => shownOf(submission, catalog))
        .filter(decided)
        .filter(({id, state}) => told[id] !== state)
}

export function withTold(told: Told, fresh: readonly Decided[]): Told {
    const next: Record<string, ShownState> = {...told}
    for (const {id, state} of fresh) next[id] = state
    return next
}
