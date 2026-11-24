import type {ListedFlag, Pin} from "../domain/flags.ts"
import type {Submission} from "../domain/flagSubmissions.ts"
import type {UploadOptions} from "./support.ts"

export interface FlagLister {
    // fresh: past the browser's copy of the last answer, for a read after the list changed.
    listFlags(fresh: boolean): Promise<ListedFlag[]>
}

export type FlagHandlers = {
    // Each connection's first message: what changed while the stream was closed is only in a read of the list.
    onOpened: () => void
    onChanged: (flag: ListedFlag) => void
}

// The flags module's stream, open while it is listened to.
export interface FlagStream {
    listenForFlags(handlers: FlagHandlers): () => void
}

export type SubmittedFlag = {
    name: string
    // None for a change of name alone.
    uploadId: string
    pin: Pin
    // A flag the account founded, for its new name or picture.
    flag?: string
}

export interface FlagSubmitter {
    // The id SubmitFlag takes, once the picture is up.
    uploadPicture(file: Blob, options?: UploadOptions): Promise<string>
    submitFlag(submitted: SubmittedFlag): Promise<Submission>
    // The newest first; none for a guest.
    mySubmissions(): Promise<Submission[]>
}

export type SubmissionRefusal = "no-session" | "guest" | "not-yours" | "waiting" | "invalid" | "off" | "too-many"

// One class and not one per reason: each is said the same way, as one line under the form.
export class SubmissionRefusedError extends Error {
    constructor(public readonly refusal: SubmissionRefusal, options?: {cause?: unknown}) {
        super(`the flags module refused the submission: ${refusal}`, options)
        this.name = "SubmissionRefusedError"
    }
}
