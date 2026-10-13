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

export type AttachmentUpload = {
    url: string
}

export type UploadOptions = {
    onProgress?: (fraction: number) => void
    signal?: AbortSignal
}

// Both answer one upload per attachment, in its order: the note is kept first, and its files go up after.
export interface FeatureSuggester {
    suggestFeature(text: string, context: PageContext, attachments?: Blob[]): Promise<AttachmentUpload[]>
}

export interface BugReporter {
    reportBug(text: string, context: PageContext, attachments?: Blob[]): Promise<AttachmentUpload[]>
}

export interface AttachmentRulesReader {
    attachmentRules(): Promise<AttachmentRules>
}

export interface AttachmentUploader {
    uploadAttachment(upload: AttachmentUpload, file: Blob, options?: UploadOptions): Promise<void>
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

export type SupportBackend = FeatureSuggester & BugReporter & PlayerReporter & AttachmentRulesReader & AttachmentUploader

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
