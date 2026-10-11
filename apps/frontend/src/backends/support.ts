export const MAX_NOTE_LENGTH = 1000

export interface FeatureSuggester {
    suggestFeature(text: string): Promise<void>
}

export type BugContext = {
    build: string
    viewportWidth: number
    viewportHeight: number
    connectionDown: boolean
}

export interface BugReporter {
    reportBug(text: string, context: BugContext): Promise<void>
}

export type ReportReason = "cheating" | "chat-abuse" | "bad-name" | "other"

export type PlayerReport = {
    name: string
    reason: ReportReason
    text: string
    messageId?: string
}

export interface PlayerReporter {
    reportPlayer(report: PlayerReport): Promise<void>
}

export type SupportBackend = FeatureSuggester & BugReporter & PlayerReporter

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

export class SupportUnknownPlayerError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("no player is shown under this name", options)
        this.name = "SupportUnknownPlayerError"
    }
}

export class SupportReportedAlreadyError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("this player was already reported today", options)
        this.name = "SupportReportedAlreadyError"
    }
}
