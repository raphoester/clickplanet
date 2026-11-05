import {NoteKind, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {PageContext, ReportReason} from "../gen/grpc/support/v1/support_pb.ts"

const KINDS: Record<NoteKind, string> = {
    [NoteKind.UNSPECIFIED]: "Unknown",
    [NoteKind.SUGGESTION]: "Idea",
    [NoteKind.BUG]: "Bug",
}

const REASONS: Record<ReportReason, string> = {
    [ReportReason.UNSPECIFIED]: "Unknown",
    [ReportReason.CHEATING]: "Cheating",
    [ReportReason.CHAT_ABUSE]: "Chat abuse",
    [ReportReason.BAD_NAME]: "Bad name",
    [ReportReason.OTHER]: "Other",
}

const OUTCOMES: Record<ReportOutcome, string> = {
    [ReportOutcome.UNSPECIFIED]: "Unknown",
    [ReportOutcome.ACTED]: "Acted",
    [ReportOutcome.DISMISSED]: "Dismissed",
}

// A value this build has no name for is one the proto added since.
export function kindLabel(kind: NoteKind): string {
    return KINDS[kind] ?? `Kind ${kind}`
}

export function reasonLabel(reason: ReportReason): string {
    return REASONS[reason] ?? `Reason ${reason}`
}

export function outcomeLabel(outcome: ReportOutcome): string {
    return OUTCOMES[outcome] ?? `Outcome ${outcome}`
}

export function countOf(n: number, noun: string): string {
    return n === 1 ? `1 ${noun}` : `${n} ${noun}s`
}

// What a form changes: "1 change", "3 changes", or none yet.
export function changesWord(n: number): string {
    return n === 0 ? "No change yet" : countOf(n, "change")
}

export type Line = {label: string, value: string}

export function deviceLines(context: PageContext | undefined, userAgent: string): Line[] {
    const lines: Line[] = []
    const add = (label: string, value: string) => {
        if (value !== "") lines.push({label, value})
    }

    if (context !== undefined) {
        add("Build", context.build)
        if (context.viewportWidth > 0) {
            add("Viewport", `${context.viewportWidth}×${context.viewportHeight} at ${context.pixelRatio}x`)
        }
        if (context.connectionDown) add("Connection", "down")
        add("GPU", [context.gpuVendor, context.gpuRenderer].filter(part => part !== "").join(" · "))
        if (context.webglVersion !== "") {
            add("WebGL", `${context.webglVersion}, textures up to ${context.maxTextureSize}px`)
        }
        if (context.contextLost) add("WebGL context", "lost")
        if (context.cores > 0) add("Cores", String(context.cores))
        if (context.memoryGb > 0) add("Memory", `${context.memoryGb} GB`)
    }
    add("Browser", userAgent)

    return lines
}

export type Media = "photo" | "video" | "file"

export function mediaOf(contentType: string): Media {
    if (contentType.startsWith("image/")) return "photo"
    if (contentType.startsWith("video/")) return "video"
    return "file"
}

export function sizeLabel(bytes: bigint | number): string {
    const n = Number(bytes)
    if (n < 1024) return `${n} B`
    if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`
    return `${(n / (1024 * 1024)).toFixed(1)} MB`
}
