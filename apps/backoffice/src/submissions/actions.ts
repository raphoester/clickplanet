import {Timestamp} from "@bufbuild/protobuf"
import {FlagsAdmin} from "../clients.ts"
import {PICTURES_OFF, Refused} from "../flags/actions.ts"
import {KeptSubmission} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag, RejectionReason} from "../gen/grpc/flags/v1/flags_pb.ts"
import {wordsOf} from "../refusal.ts"

// The backend's most for a rejection's note.
export const MAX_NOTE = 500

const GONE = "This submission is no longer kept. Reload the list."

export async function approve(flags: FlagsAdmin, submissionId: string): Promise<KeptSubmission> {
    return answered((await flags.approveSubmission({submissionId})).submission)
}

export async function reject(flags: FlagsAdmin, submissionId: string, reason: RejectionReason, note: string): Promise<KeptSubmission> {
    return answered((await flags.rejectSubmission({submissionId, rejection: {reason, note: note.trim()}})).submission)
}

export async function openSubmitted(flags: FlagsAdmin, submissionId: string, endsAt: Date): Promise<Flag> {
    const {flag} = await flags.openSubmittedFlag({submissionId, endsAt: Timestamp.fromDate(endsAt)})
    if (flag === undefined) throw new Refused("The server answered no flag. Reload the list.")
    return flag
}

export function approveRefusalOf(err: unknown): string {
    return refused(err) ?? wordsOf(err, {notFound: GONE, failedPrecondition: `It is decided already, or ${lower(PICTURES_OFF)}`})
}

export function rejectRefusalOf(err: unknown): string {
    return refused(err) ?? wordsOf(err, {notFound: GONE, failedPrecondition: "It is decided already. Reload the list."})
}

export function openRefusalOf(err: unknown): string {
    return refused(err) ?? wordsOf(err, {
        notFound: GONE,
        failedPrecondition: "It is not an approved new flag: opened already, or a change. Reload the list.",
    })
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

function lower(sentence: string): string {
    return sentence.charAt(0).toLowerCase() + sentence.slice(1)
}

function answered(submission: KeptSubmission | undefined): KeptSubmission {
    if (submission === undefined) throw new Refused("The server answered no submission. Reload the list.")
    return submission
}
