export const CHAT_SIZE_STORAGE_KEY = 'clickplanet-chat-size'

export type ChatSize = {
    width: number
    height: number
}

export const RESIZE_EDGES = ["top", "left", "corner"] as const

export type ResizeEdge = typeof RESIZE_EDGES[number]

export function parseStoredSize(raw: string | null | undefined): ChatSize | undefined {
    if (!raw) return undefined

    let parsed: unknown
    try {
        parsed = JSON.parse(raw)
    } catch {
        return undefined
    }

    if (typeof parsed !== "object" || parsed === null) return undefined

    const {width, height} = parsed as {width?: unknown, height?: unknown}
    if (!isLength(width) || !isLength(height)) return undefined

    return {width, height}
}

export function draggedSize(edge: ResizeEdge, start: ChatSize, dx: number, dy: number): ChatSize {
    return {
        width: edge === "top" ? start.width : start.width - dx,
        height: edge === "left" ? start.height : start.height - dy,
    }
}

function isLength(value: unknown): value is number {
    return typeof value === "number" && Number.isFinite(value) && value > 0
}
