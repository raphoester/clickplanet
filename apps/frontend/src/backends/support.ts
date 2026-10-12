export const MAX_NOTE_LENGTH = 1000

export interface FeatureSuggester {
    suggestFeature(text: string): Promise<void>
}

export type SupportBackend = FeatureSuggester

export class SupportRateLimitedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("too many sent to the support service", options)
        this.name = "SupportRateLimitedError"
    }
}

export class SupportRejectedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the support service refused what was sent", options)
        this.name = "SupportRejectedError"
    }
}

export class SupportNoSessionError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("no session to reach the support service with", options)
        this.name = "SupportNoSessionError"
    }
}
