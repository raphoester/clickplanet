import type {FlagPicture, Pin} from "./flags.ts"

export type SubmissionState = "waiting" | "approved" | "rejected" | "opened"

export type RejectionReason = "not-a-place" | "exists" | "unreadable" | "hateful" | "other"

export type Rejection = {reason: RejectionReason, note: string}

// A flag the player submitted, as its player reads it; times in milliseconds since the epoch.
export type Submission = {
    id: string
    name: string
    pin?: Pin
    // A new name or picture for a flag already opened.
    change: boolean
    // The flag a change is for, or the one opened from it; empty until then.
    flag: string
    state: SubmissionState
    submittedAt: number
    decidedAt: number
    rejection?: Rejection
    picture?: FlagPicture
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

export function waits(submissions: readonly Submission[]): boolean {
    return submissions.some((submission) => submission.state === "waiting")
}

// The flags the account founded, the newest first: each opened from one of its submissions, or changed by one.
export function foundedFlags(submissions: readonly Submission[]): string[] {
    const founded = submissions
        .filter(({state, flag, change}) => flag !== "" && (state === "opened" || (change && state === "approved")))
        .map(({flag}) => flag)
    return [...new Set(founded)]
}

export type Decided = Submission & {state: Exclude<SubmissionState, "waiting">}

export function decided(submission: Submission): submission is Decided {
    return submission.state !== "waiting"
}

// The state each submission was last told in, by id.
export type Told = Readonly<Record<string, SubmissionState>>

// The decisions not told yet, the newest first: a submission approved, then opened, is told twice.
export function untold(submissions: readonly Submission[], told: Told): Decided[] {
    return submissions.filter(decided).filter((submission) => told[submission.id] !== submission.state)
}

export function withTold(told: Told, submissions: readonly Submission[]): Told {
    const next: Record<string, SubmissionState> = {...told}
    for (const submission of submissions) if (decided(submission)) next[submission.id] = submission.state
    return next
}
