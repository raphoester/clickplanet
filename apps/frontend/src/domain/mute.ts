const UNITS: [name: string, seconds: number][] = [
    ["week", 7 * 24 * 3600],
    ["day", 24 * 3600],
    ["hour", 3600],
    ["minute", 60],
    ["second", 1],
]

import {MuteKind} from "../backends/chat.ts"

export function describeMute(seconds: number, kind: MuteKind = 'chat'): string {
    if (kind === 'gifs') return `may not post GIFs for ${muteLength(seconds)}`
    return `has been muted for ${muteLength(seconds)}`
}

export function muteLength(seconds: number): string {
    const [unit, size] = UNITS.find(([, size]) => seconds >= size && seconds % size === 0) ?? ["second", 1]
    const count = Math.round(seconds / size)
    return count === 1 ? `one ${unit}` : `${count} ${unit}s`
}
