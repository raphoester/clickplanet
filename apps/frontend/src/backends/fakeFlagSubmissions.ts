import type {FlagPicture} from "../domain/flags.ts"
import {
    flagNameOf,
    foundedFlags,
    isValidFlagName,
    pictureFileRefusal,
    RejectionReason,
    Submission,
} from "../domain/flagSubmissions.ts"
import {FakeFlagsBackend, NameRefusal} from "./fakeFlagsBackend.ts"
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

    public async submitFlag({name, uploadId, pin, flag}: SubmittedFlag): Promise<Submission> {
        await this.answer()
        const upload = this.uploads.get(uploadId)
        if (!isValidFlagName(name) || Math.abs(pin.latitude) > 90 || Math.abs(pin.longitude) > 180) {
            throw new SubmissionRefusedError("invalid")
        }
        if (flag === undefined ? upload === undefined : uploadId !== "" && upload === undefined) {
            throw new SubmissionRefusedError("invalid")
        }
        const submissions = this.submissions()
        if (flag !== undefined && !foundedFlags(submissions).includes(flag)) throw new SubmissionRefusedError("not-yours")
        if (submissions.some(({state}) => state === "waiting")) throw new SubmissionRefusedError("waiting")

        const at = this.now()
        const submission: Submission = {
            id: crypto.randomUUID(),
            name: flagNameOf(name),
            pin,
            change: flag !== undefined,
            flag: flag ?? "",
            state: "waiting",
            submittedAt: at,
            decidedAt: at,
        }
        this.uploads.delete(uploadId)
        this.kept.unshift({submission, upload})
        return {...submission}
    }

    public async mySubmissions(): Promise<Submission[]> {
        await this.wait(this.delayMs / 4)
        if (!await this.linked()) return []
        return this.submissions().map((submission) => ({...submission}))
    }

    public async approve(): Promise<string> {
        const kept = this.waiting()
        if (!kept) return "no submission waits: send one from the flag picker"
        const picture = kept.upload && await this.paint(kept.upload)
        this.decide(kept, {state: "approved", picture})
        if (kept.submission.change) {
            this.flags.change(kept.submission.flag, kept.submission.name, picture)
            this.flags.reopen()
            return `the change to ${kept.submission.name} is on its flag`
        }
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
        const kept = this.kept.find(({submission}) => submission.state === "approved" && !submission.change)
        if (!kept) return "no new flag is approved: approveSubmission() approves the one that waits"
        const {name} = kept.submission
        const opened = await this.openApproved(kept.submission.id, days)
        if ("refused" in opened) return `${name} cannot open: ${opened.refused}`
        return `${name} is open as ${opened.code} for ${days} days`
    }

    // An approved new flag opened, as its first payment would open it.
    public async openApproved(id: string, days: number): Promise<{code: string} | {refused: NameRefusal | "not approved"}> {
        const kept = this.kept.find(({submission}) => submission.id === id)
        if (!kept || kept.submission.state !== "approved" || kept.submission.change) return {refused: "not approved"}
        const {name, pin, picture} = kept.submission
        const opened = await this.flags.open(name, days, {picture, pin})
        if ("refused" in opened) return opened
        this.decide(kept, {state: "opened", flag: opened.code})
        this.flags.reopen()
        return opened
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
