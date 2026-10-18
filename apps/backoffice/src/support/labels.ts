import {NoteKind} from "../gen/grpc/support/v1/admin_pb.ts"
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

// A value this build has no name for is one the proto added since.
export function kindLabel(kind: NoteKind): string {
    return KINDS[kind] ?? `Kind ${kind}`
}

export function reasonLabel(reason: ReportReason): string {
    return REASONS[reason] ?? `Reason ${reason}`
}

export function countOf(n: number, noun: string): string {
    return n === 1 ? `1 ${noun}` : `${n} ${noun}s`
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
