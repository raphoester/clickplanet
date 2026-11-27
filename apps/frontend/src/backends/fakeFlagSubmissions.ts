import type {FlagPicture} from "../domain/flags.ts"
import {flagNameOf, isValidFlagName, MyFlags, pictureFileRefusal, RejectionReason, Submission} from "../domain/flagSubmissions.ts"
import {FakeFlagsBackend, NameRefusal} from "./fakeFlagsBackend.ts"
import {AppealRefusedError, FlagSubmitter, SubmissionRefusal, SubmissionRefusedError, SubmittedFlag} from "./flags.ts"
import {UploadOptions} from "./support.ts"

const ANSWER_DELAY_MS = 400
const UPLOAD_STEPS = 8

// The server's drawing of an upload, as a sold flag's picture.
export type UploadPainter = (upload: Blob) => Promise<FlagPicture>

// paidDays: the time paid while it waited, held until its approval opens it.
type Kept = {submission: Submission, upload?: Blob, paidDays?: number}

// The fake's judge rejects, until its player asks a person: the operator's rejection after is final.
type Rejecter = "judge" | "operator"

// The flags module's submissions in memory, and the operator's verdicts from the console: approveSubmission(),
// rejectSubmission(reason, note) (the judge's, or the operator's once its player asked a person), openSubmission().
// After each verdict the flags stream opens again, so the page reads it at once, as it would at its next opening.
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
        if (replaces !== undefined
            && (waiting?.submission.id !== replaces || waiting.paidDays !== undefined || waiting.submission.appealed)) {
            throw new SubmissionRefusedError("decided")
        }
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
            paid: false,
        }
        this.uploads.delete(uploadId)
        if (waiting !== undefined) this.kept.splice(this.kept.indexOf(waiting), 1)
        this.kept.unshift({submission, upload})
        return {...submission}
    }

    public async myFlags(): Promise<MyFlags> {
        await this.wait(this.delayMs / 4)
        if (!await this.linked()) return {submissions: [], held: []}
        return {
            submissions: this.kept.map(({submission, paidDays}) => {
                const paid = paidDays !== undefined && (submission.state === "waiting" || submission.state === "approved")
                return {...submission, paid}
            }),
            held: this.flags.heldFlags(),
        }
    }

    // As the shop's payment for a flag that waits: its time is held until the approval.
    public hold(id: string, days: number): boolean {
        const kept = this.kept.find(({submission}) => submission.id === id)
        if (!kept || kept.submission.state !== "waiting" || kept.paidDays !== undefined) return false
        kept.paidDays = days
        return true
    }

    public async approve(): Promise<string> {
        const kept = this.waiting()
        if (!kept) return "no submission waits: send one from the flag picker"
        const picture = kept.upload && await this.paint(kept.upload)
        this.decide(kept, {state: "approved", picture})
        this.flags.reopen()
        if (kept.paidDays === undefined) return `${kept.submission.name} is approved: openSubmission() opens it, as a payment would`
        const {name} = kept.submission
        const opened = await this.openApproved(kept.submission.id, kept.paidDays)
        if ("refused" in opened) return `${name} is approved and paid, but cannot open: ${opened.refused}`
        return `${name} is approved and opens with its paid time, as ${opened.code}`
    }

    // The judge's rejection, or the operator's once a person was asked to look again.
    public reject(reason: RejectionReason = "other", note = ""): string {
        const kept = this.waiting()
        if (!kept) return "no submission waits: send one from the flag picker"
        const refunded = kept.paidDays !== undefined
        const by: Rejecter = kept.submission.appealed ? "operator" : "judge"
        kept.paidDays = undefined
        this.decide(kept, {state: "rejected", rejection: {reason, note}, rejectedByJudge: by === "judge"})
        this.flags.reopen()
        return `${kept.submission.name} is rejected by the ${by}: ${reason}${refunded ? ", and its payment refunded" : ""}`
    }

    // Once per submission, a rejection the judge made, while no other waits.
    public async appealSubmission(id: string): Promise<Submission> {
        await this.answer()
        const kept = this.kept.find(({submission}) => submission.id === id)
        if (!kept || kept.submission.state !== "rejected" || !kept.submission.rejectedByJudge) {
            throw new AppealRefusedError("refused")
        }
        if (this.waiting() !== undefined) throw new AppealRefusedError("waiting")
        const asked: Submission = {...kept.submission, state: "waiting", appealed: true, decidedAt: this.now()}
        delete asked.rejection
        delete asked.rejectedByJudge
        kept.submission = asked
        return {...asked}
    }

    public async open(days = 30): Promise<string> {
        const kept = this.kept.find(({submission}) => submission.state === "approved")
        if (!kept) return "no flag is approved: approveSubmission() approves the one that waits"
        const {name} = kept.submission
        const opened = await this.openApproved(kept.submission.id, days)
        if ("refused" in opened) return `${name} cannot open: ${opened.refused}`
        return `${name} is open as ${opened.code} for ${days} days`
    }

    // An approved flag opened, as its first payment would open it.
    public async openApproved(id: string, days: number): Promise<{code: string} | {refused: NameRefusal | "not approved"}> {
        const kept = this.kept.find(({submission}) => submission.id === id)
        if (!kept || kept.submission.state !== "approved") return {refused: "not approved"}
        const {name, pin, picture} = kept.submission
        const opened = await this.flags.open(name, days, {picture, pin, held: true})
        if ("refused" in opened) return opened
        kept.paidDays = undefined
        this.decide(kept, {state: "opened", flag: opened.code})
        this.flags.reopen()
        return opened
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
