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

export type AttachmentRules = {
    maxPerNote: number
    maxBytes: number
    contentTypes: string[]
}

export const NO_ATTACHMENTS: AttachmentRules = {maxPerNote: 0, maxBytes: 0, contentTypes: []}

export type UploadOptions = {
    onProgress?: (fraction: number) => void
    signal?: AbortSignal
}

// Each attachment id is a file already uploaded through AttachmentUploader.
export interface FeatureSuggester {
    suggestFeature(text: string, context: PageContext, attachmentIds?: string[]): Promise<void>
}

export interface BugReporter {
    reportBug(text: string, context: PageContext, attachmentIds?: string[]): Promise<void>
}

export interface AttachmentRulesReader {
    attachmentRules(): Promise<AttachmentRules>
}

export interface AttachmentUploader {
    // Answers the id a note names the file by, once it is up.
    uploadAttachment(file: Blob, options?: UploadOptions): Promise<string>
}

export type ReportReason = "cheating" | "chat-abuse" | "bad-name" | "other"

export type PlayerReport = {
    accountId: string
    reason: ReportReason
    text: string
    messageId?: string
}

export interface PlayerReporter {
    reportPlayer(report: PlayerReport): Promise<void>
}

export type PlaceReportReason = "hate" | "sexual" | "other"

// A drawing on the map, by the tile the player picked in it.
export type PlaceReport = {
    tileId: number
    reason: PlaceReportReason
    text: string
}

export interface PlaceReporter {
    reportPlace(report: PlaceReport): Promise<void>
}

// What the feedback form asks for: a note of either kind, and its files.
export type FeedbackBackend = FeatureSuggester & BugReporter & AttachmentRulesReader & AttachmentUploader

export type SupportBackend = FeedbackBackend & PlayerReporter & PlaceReporter

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

export class SupportReportedAlreadyError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("this player was already reported today", options)
        this.name = "SupportReportedAlreadyError"
    }
}
