import type {FlagPicture} from "../domain/flags.ts"
import {flagNameOf, isValidFlagName, pictureFileRefusal, RejectionReason, Submission} from "../domain/flagSubmissions.ts"
import {FakeFlagsBackend} from "./fakeFlagsBackend.ts"
import {FlagSubmitter, SubmissionRefusal, SubmissionRefusedError, SubmittedFlag} from "./flags.ts"
import {UploadOptions} from "./support.ts"

const ANSWER_DELAY_MS = 400
const UPLOAD_STEPS = 8

// The server's drawing of an upload, as a sold flag's picture.
export type UploadPainter = (upload: Blob) => Promise<FlagPicture>

type Kept = {submission: Submission, upload?: Blob}

// The flags module's submissions in memory, and the operator's verdicts from the console: approveSubmission(),
// rejectSubmission(reason, note), openSubmission(). After each verdict the flags stream opens again, so the page reads
// it at once, as it would at its next opening.
export class FakeFlagSubmissions implements FlagSubmitter {
    private readonly kept: Kept[] = []
    private readonly uploads = new Map<string, Blob>()
    private refusal?: SubmissionRefusal | "upload-failed"

    constructor(
        private readonly flags: FakeFlagsBackend,
        private readonly linked: () => Promise<boolean>,
        private readonly paint: UploadPainter,
        private readonly delayMs = ANSWER_DELAY_MS,
        private readonly now: () => number = Date.now,
    ) {
    }

    public refuse(refusal?: SubmissionRefusal | "upload-failed"): string {
        this.refusal = refusal
        return refusal ? `the next submissions are refused: ${refusal}` : "the next submissions go through"
    }

    public async uploadPicture(file: Blob, {onProgress, signal}: UploadOptions = {}): Promise<string> {
        await this.answer()
        if (pictureFileRefusal(file)) throw new SubmissionRefusedError("invalid")
        for (let step = 1; step <= UPLOAD_STEPS; step++) {
            await this.wait(this.delayMs / 2)
            if (signal?.aborted) throw new Error("the upload was called off")
            if (this.refusal === "upload-failed" && step === UPLOAD_STEPS / 2) throw new Error("the fake upload failed")
            onProgress?.(step / UPLOAD_STEPS)
        }
        const id = `fake-upload-${this.uploads.size}`
        this.uploads.set(id, file)
        return id
    }

    // One that replaces the submission that waits takes its place; one the operator decided never changes.
    public async submitFlag({name, uploadId, pin, replaces}: SubmittedFlag): Promise<Submission> {
        await this.answer()
        const upload = this.uploads.get(uploadId)
        if (!isValidFlagName(name) || Math.abs(pin.latitude) > 90 || Math.abs(pin.longitude) > 180 || upload === undefined) {
            throw new SubmissionRefusedError("invalid")
        }
        const waiting = this.waiting()
        if (replaces !== undefined && waiting?.submission.id !== replaces) throw new SubmissionRefusedError("decided")
        if (replaces === undefined && waiting !== undefined) throw new SubmissionRefusedError("waiting")

        const at = this.now()
        const submission: Submission = {
            id: crypto.randomUUID(),
            name: flagNameOf(name),
            pin,
            flag: "",
            state: "waiting",
            submittedAt: at,
            decidedAt: at,
        }
        this.uploads.delete(uploadId)
        if (waiting !== undefined) this.kept.splice(this.kept.indexOf(waiting), 1)
        this.kept.unshift({submission, upload})
        return {...submission}
    }

    public async mySubmissions(): Promise<Submission[]> {
        await this.wait(this.delayMs / 4)
        if (!await this.linked()) return []
        return this.submissions().map((submission) => {
            const closeNote = submission.state === "opened" ? this.flags.closeNoteOf(submission.flag) : undefined
            return closeNote === undefined ? {...submission} : {...submission, closeNote}
        })
    }

    public async approve(): Promise<string> {
        const kept = this.waiting()
        if (!kept) return "no submission waits: send one from the flag picker"
        const picture = kept.upload && await this.paint(kept.upload)
        this.decide(kept, {state: "approved", picture})
        this.flags.reopen()
        return `${kept.submission.name} is approved: openSubmission() opens it, as a payment would`
    }

    public reject(reason: RejectionReason = "other", note = ""): string {
        const kept = this.waiting()
        if (!kept) return "no submission waits: send one from the flag picker"
        this.decide(kept, {state: "rejected", rejection: {reason, note}})
        this.flags.reopen()
        return `${kept.submission.name} is rejected: ${reason}`
    }

    public async open(days = 30): Promise<string> {
        const kept = this.kept.find(({submission}) => submission.state === "approved")
        if (!kept) return "no flag is approved: approveSubmission() approves the one that waits"
        const {name, pin, picture} = kept.submission
        const opened = await this.flags.open(name, days, {picture, pin})
        if ("refused" in opened) return `${name} cannot open: ${opened.refused}`
        this.decide(kept, {state: "opened", flag: opened.code})
        this.flags.reopen()
        return `${name} is open as ${opened.code} for ${days} days`
    }

    private submissions(): Submission[] {
        return this.kept.map(({submission}) => submission)
    }

    private waiting(): Kept | undefined {
        return this.kept.find(({submission}) => submission.state === "waiting")
    }

    private decide(kept: Kept, decision: Partial<Submission>) {
        kept.submission = {...kept.submission, ...decision, decidedAt: this.now()}
    }

    private async answer(): Promise<void> {
        await this.wait(this.delayMs)
        if (!await this.linked()) throw new SubmissionRefusedError("guest")
        if (this.refusal !== undefined && this.refusal !== "upload-failed") throw new SubmissionRefusedError(this.refusal)
    }

    private wait(ms: number): Promise<void> {
        return new Promise((resolve) => setTimeout(resolve, ms))
    }
}
