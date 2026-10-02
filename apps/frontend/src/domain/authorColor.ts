export function authorHue(authorName: string): number {
    return hash(authorName) % 360
}

const FNV_OFFSET = 0x811c9dc5
const FNV_PRIME = 0x01000193

function hash(key: string): number {
    let value = FNV_OFFSET
    for (const rune of key) {
        value ^= rune.codePointAt(0)!
        value = Math.imul(value, FNV_PRIME)
    }
    return value >>> 0
}
