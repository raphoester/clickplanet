import {
    AttachmentRules,
    PageContext,
    PlayerReport,
    SupportBackend,
    SupportNoSessionError,
    SupportRateLimitedError,
    SupportRejectedError,
    SupportReportedAlreadyError,
    UploadOptions,
} from "./support.ts";

export type FakeSupportRefusal = "rate-limited" | "rejected" | "no-session" | "reported-already" | "failed" | "upload-failed"

type FakeNoteKind = "suggestion" | "bug"

export type FakeSupportNote =
    | {kind: FakeNoteKind, text: string, context: PageContext, attachmentIds: string[]}
    | {kind: "report", report: PlayerReport}

const ANSWER_DELAY_MS = 400
const UPLOAD_STEPS = 8

const FAKE_RULES: AttachmentRules = {
    maxPerNote: 10,
    maxBytes: 50 * 1024 * 1024,
    contentTypes: [
        "image/jpeg", "image/png", "image/webp", "image/gif", "image/heic", "image/heif",
        "video/mp4", "video/quicktime", "video/webm",
    ],
}

export class FakeSupportBackend implements SupportBackend {
    public readonly sent: FakeSupportNote[] = []

    // A file's id is fake-file- and its place here.
    public readonly uploaded: Blob[] = []

    public rules: AttachmentRules = FAKE_RULES

    private refusal?: FakeSupportRefusal

    constructor(private readonly delayMs = ANSWER_DELAY_MS) {
    }

    public refuse(refusal?: FakeSupportRefusal): string {
        this.refusal = refusal
        return refusal ? `the next sends are refused: ${refusal}` : "the next sends go through"
    }

    public suggestFeature(text: string, context: PageContext, attachmentIds: string[] = []): Promise<void> {
        return this.keep("suggestion", text, context, attachmentIds)
    }

    public reportBug(text: string, context: PageContext, attachmentIds: string[] = []): Promise<void> {
        return this.keep("bug", text, context, attachmentIds)
    }

    public async attachmentRules(): Promise<AttachmentRules> {
        return this.rules
    }

    public async uploadAttachment(file: Blob, {onProgress, signal}: UploadOptions = {}): Promise<string> {
        for (let step = 1; step <= UPLOAD_STEPS; step++) {
            await new Promise((resolve) => setTimeout(resolve, this.delayMs / 2))
            if (signal?.aborted) throw new Error("the upload was called off")
            if (this.refusal === "upload-failed" && step === UPLOAD_STEPS / 2) throw new Error("the fake upload failed")
            onProgress?.(step / UPLOAD_STEPS)
        }
        return `fake-file-${this.uploaded.push(file) - 1}`
    }

    private async keep(kind: FakeNoteKind, text: string, context: PageContext, attachmentIds: string[]): Promise<void> {
        await this.answer()
        this.sent.push({kind, text, context, attachmentIds})
    }

    public async reportPlayer(report: PlayerReport): Promise<void> {
        await this.answer()
        this.sent.push({kind: "report", report})
    }

    private async answer(): Promise<void> {
        await new Promise((resolve) => setTimeout(resolve, this.delayMs))

        switch (this.refusal) {
            case "rate-limited":
                throw new SupportRateLimitedError()
            case "rejected":
                throw new SupportRejectedError()
            case "no-session":
                throw new SupportNoSessionError()
            case "reported-already":
                throw new SupportReportedAlreadyError()
            case "failed":
                throw new Error("the fake support backend failed")
        }
    }
}
