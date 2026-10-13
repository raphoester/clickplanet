import {
    AttachmentRules,
    AttachmentUpload,
    PageContext,
    SupportBackend,
    SupportNoSessionError,
    SupportRateLimitedError,
    SupportRejectedError,
    UploadOptions,
} from "./support.ts";

export type FakeSupportRefusal = "rate-limited" | "rejected" | "no-session" | "failed" | "upload-failed"

export type FakeSupportNote = {
    kind: "suggestion" | "bug"
    text: string
    context: PageContext
    attachments: Blob[]
    // The place in attachments of each file that was uploaded.
    uploaded: number[]
}

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

    public rules: AttachmentRules = FAKE_RULES

    private refusal?: FakeSupportRefusal

    constructor(private readonly delayMs = ANSWER_DELAY_MS) {
    }

    public refuse(refusal?: FakeSupportRefusal): string {
        this.refusal = refusal
        return refusal ? `the next sends are refused: ${refusal}` : "the next sends go through"
    }

    public suggestFeature(text: string, context: PageContext, attachments: Blob[] = []): Promise<AttachmentUpload[]> {
        return this.keep("suggestion", text, context, attachments)
    }

    public reportBug(text: string, context: PageContext, attachments: Blob[] = []): Promise<AttachmentUpload[]> {
        return this.keep("bug", text, context, attachments)
    }

    public async attachmentRules(): Promise<AttachmentRules> {
        return this.rules
    }

    public async uploadAttachment(upload: AttachmentUpload, _file: Blob, {onProgress, signal}: UploadOptions = {}): Promise<void> {
        const [note, place] = upload.url.replace("fake://", "").split("/").map(Number)

        for (let step = 1; step <= UPLOAD_STEPS; step++) {
            await new Promise((resolve) => setTimeout(resolve, this.delayMs / 2))
            if (signal?.aborted) throw new Error("the upload was called off")
            if (this.refusal === "upload-failed" && step === UPLOAD_STEPS / 2) throw new Error("the fake upload failed")
            onProgress?.(step / UPLOAD_STEPS)
        }
        this.sent[note].uploaded.push(place)
    }

    private async keep(
        kind: FakeSupportNote["kind"], text: string, context: PageContext, attachments: Blob[],
    ): Promise<AttachmentUpload[]> {
        await this.answer()

        const note = this.sent.push({kind, text, context, attachments, uploaded: []}) - 1
        return attachments.map((_, place) => ({url: `fake://${note}/${place}`}))
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
            case "failed":
                throw new Error("the fake support backend failed")
        }
    }
}
