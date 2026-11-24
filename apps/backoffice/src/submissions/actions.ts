import {Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {FlagsAdmin} from "../clients.ts"
import {Refused} from "../flags/actions.ts"
import {Place} from "../flags/pin.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag, RejectionReason} from "../gen/grpc/flags/v1/flags_pb.ts"
import {localized, wordsOf} from "../refusal.ts"

// The backend's most for a rejection's note.
export const MAX_NOTE = 500

const GONE = "This submission is no longer kept. Reload the list."

// At the player's capital, or at `moved`.
export async function approve(flags: FlagsAdmin, submissionId: string, moved?: Place): Promise<KeptSubmission> {
    const request = moved === undefined ? {submissionId} : {submissionId, pin: moved}
    return answered((await flags.approveSubmission(request)).submission)
}

export async function reject(flags: FlagsAdmin, submissionId: string, reason: RejectionReason, note: string): Promise<KeptSubmission> {
    return answered((await flags.rejectSubmission({submissionId, rejection: {reason, note: note.trim()}})).submission)
}

export async function openSubmitted(flags: FlagsAdmin, submissionId: string, endsAt: Date): Promise<Flag> {
    const {flag} = await flags.openSubmittedFlag({submissionId, endsAt: Timestamp.fromDate(endsAt)})
    if (flag === undefined) throw new Refused("The server answered no flag. Reload the list.")
    return flag
}

// A failed_precondition is quoted: decided already, a name another flag took since, or the pictures off.
export function approveRefusalOf(err: unknown): string {
    return refused(err) ?? quoted(err) ?? wordsOf(err, {notFound: GONE})
}

export function rejectRefusalOf(err: unknown): string {
    return refused(err) ?? wordsOf(err, {notFound: GONE, failedPrecondition: "It is decided already. Reload the list."})
}

// A failed_precondition is quoted: opened already, a change, or a name another flag took since.
export function openRefusalOf(err: unknown): string {
    return refused(err) ?? quoted(err) ?? wordsOf(err, {notFound: GONE})
}

export function lengthOf(note: string): number {
    return [...note.trim()].length
}

// A rejection the server takes: a reason, and a note of MAX_NOTE at most, which Other needs.
export function rejectionReady(reason: RejectionReason | undefined, note: string): boolean {
    if (reason === undefined || lengthOf(note) > MAX_NOTE) return false
    return reason !== RejectionReason.OTHER || lengthOf(note) > 0
}

function refused(err: unknown): string | undefined {
    return err instanceof Refused ? err.message : undefined
}

function quoted(err: unknown): string | undefined {
    const error = ConnectError.from(err)
    return error.code === Code.FailedPrecondition ? `The server refused it: ${localized(error.rawMessage)}` : undefined
}

function answered(submission: KeptSubmission | undefined): KeptSubmission {
    if (submission === undefined) throw new Refused("The server answered no submission. Reload the list.")
    return submission
}
