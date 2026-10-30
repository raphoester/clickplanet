import {Code, ConnectError} from "@connectrpc/connect"
import {ClipsAdmin} from "../clients.ts"
import {wordsOf} from "../refusal.ts"

export type Put = (url: string, file: Blob, contentType: string, onSent: (bytes: number) => void) => Promise<void>

export const ACCEPTED = "video/mp4,video/quicktime,video/webm"
export const VIDEOS_OFF = "Videos are off on the server."
export const NOT_IN_BUCKET = "The file is not in the bucket. Upload it again."
export const VIDEO_GONE = "This video is no longer kept."

// `resend`: the bucket does not hold the file, so the next try sends it again.
export class Refused extends Error {
    constructor(words: string, readonly resend: boolean) {
        super(words)
    }
}

export function nameOf(fileName: string): string {
    const dot = fileName.lastIndexOf(".")
    return dot > 0 ? fileName.slice(0, dot) : fileName
}

// The video's id once the bucket holds the file.
export async function sendFile(clips: ClipsAdmin, put: Put, file: File, onSent: (bytes: number) => void): Promise<string> {
    let link
    try {
        link = await clips.createVideoUpload({contentType: file.type, sizeBytes: BigInt(file.size)})
    } catch (err) {
        throw new Refused(wordsOf(err, {failedPrecondition: VIDEOS_OFF}), true)
    }
    try {
        await put(link.uploadUrl, file, file.type, onSent)
    } catch (err) {
        throw new Refused(`The upload failed: ${err instanceof Error ? err.message : String(err)}.`, true)
    }
    return link.videoId
}

export async function keepVideo(clips: ClipsAdmin, videoId: string, name: string, caption: string): Promise<void> {
    try {
        await clips.keepVideo({videoId, name: name.trim(), caption: caption.trim()})
    } catch (err) {
        const resend = ConnectError.from(err).code === Code.FailedPrecondition
        throw new Refused(wordsOf(err, {failedPrecondition: NOT_IN_BUCKET}), resend)
    }
}

export function removeRefusalOf(err: unknown): string {
    return wordsOf(err, {notFound: VIDEO_GONE})
}

export function percentOf(sent: number, size: number): number {
    return size === 0 ? 100 : Math.min(100, Math.floor(sent * 100 / size))
}

// The old copy command where the page has no clipboard: a page served over http, an old Safari.
export function copyText(text: string): Promise<boolean> {
    if (navigator.clipboard === undefined) return Promise.resolve(copyByCommand(text))
    return navigator.clipboard.writeText(text).then(() => true, () => copyByCommand(text))
}

function copyByCommand(text: string): boolean {
    const area = document.createElement("textarea")
    area.value = text
    area.readOnly = true
    area.style.position = "fixed"
    area.style.opacity = "0"
    document.body.append(area)
    area.select()
    area.setSelectionRange(0, text.length)
    try {
        return document.execCommand("copy")
    } catch {
        return false
    } finally {
        area.remove()
    }
}
