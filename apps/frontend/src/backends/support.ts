export const MAX_NOTE_LENGTH = 1000

export type PageContext = {
    build: string
    viewportWidth: number
    viewportHeight: number
    pixelRatio: number
    connectionDown: boolean
    gpuVendor: string
    gpuRenderer: string
    webglVersion: string
    maxTextureSize: number
    contextLost: boolean
    cores: number
    memoryGb: number
}

export interface FeatureSuggester {
    suggestFeature(text: string, context: PageContext): Promise<void>
}

export interface BugReporter {
    reportBug(text: string, context: PageContext): Promise<void>
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
