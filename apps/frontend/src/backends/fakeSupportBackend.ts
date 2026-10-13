import {
    PageContext,
    PlayerReport,
    SupportBackend,
    SupportNoSessionError,
    SupportRateLimitedError,
    SupportRejectedError,
    SupportReportedAlreadyError,
} from "./support.ts";

export type FakeSupportRefusal = "rate-limited" | "rejected" | "no-session" | "reported-already" | "failed"

export type FakeSupportNote =
    | {kind: "suggestion" | "bug", text: string, context: PageContext}
    | {kind: "report", report: PlayerReport}

const ANSWER_DELAY_MS = 400

export class FakeSupportBackend implements SupportBackend {
    public readonly sent: FakeSupportNote[] = []

    private refusal?: FakeSupportRefusal

    constructor(private readonly delayMs = ANSWER_DELAY_MS) {
    }

    public refuse(refusal?: FakeSupportRefusal): string {
        this.refusal = refusal
        return refusal ? `the next sends are refused: ${refusal}` : "the next sends go through"
    }

    public async suggestFeature(text: string, context: PageContext): Promise<void> {
        await this.answer()
        this.sent.push({kind: "suggestion", text, context})
    }

    public async reportBug(text: string, context: PageContext): Promise<void> {
        await this.answer()
        this.sent.push({kind: "bug", text, context})
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
