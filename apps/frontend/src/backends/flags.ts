import type {ListedFlag, Pin} from "../domain/flags.ts"
import type {MyFlags, Submission} from "../domain/flagSubmissions.ts"
import type {UploadOptions} from "./support.ts"

export interface FlagLister {
    // fresh: past the browser's copy of the last answer, for a read after the list changed.
    listFlags(fresh: boolean): Promise<ListedFlag[]>
}

export type FlagHandlers = {
    // Each connection's first message: what changed while the stream was closed is only in a read of the list.
    onOpened: () => void
    // A flag opened, closed or renamed, as it is now.
    onChanged: (flag: ListedFlag) => void
}

// The flags module's stream, open while it is listened to.
export interface FlagStream {
    listenForFlags(handlers: FlagHandlers): () => void
}

export type SubmittedFlag = {
    name: string
    uploadId: string
    pin: Pin
    // The account's submission that waits, which this one replaces.
    replaces?: string
}

export interface FlagSubmitter {
    // The id SubmitFlag takes, once the picture is up.
    uploadPicture(file: Blob, options?: UploadOptions): Promise<string>
    submitFlag(submitted: SubmittedFlag): Promise<Submission>
    // The submissions, the newest first, and the flags held; none for a guest.
    myFlags(): Promise<MyFlags>
    // Asks a person to look again at the judge's rejection: the submission waits again.
    appealSubmission(id: string): Promise<Submission>
}

// One waits already, or the server refused it for good: the operator rejected it, or its picture is gone.
export type AppealRefusal = "waiting" | "refused"

export class AppealRefusedError extends Error {
    constructor(public readonly refusal: AppealRefusal, options?: {cause?: unknown}) {
        super(`the flags module refused the appeal: ${refusal}`, options)
        this.name = "AppealRefusedError"
    }
}

export type SubmissionRefusal = "no-session" | "guest" | "decided" | "waiting" | "invalid" | "off" | "too-many"

// One class and not one per reason: each is said the same way, as one line under the form.
export class SubmissionRefusedError extends Error {
    constructor(public readonly refusal: SubmissionRefusal, options?: {cause?: unknown}) {
        super(`the flags module refused the submission: ${refusal}`, options)
        this.name = "SubmissionRefusedError"
    }
}
