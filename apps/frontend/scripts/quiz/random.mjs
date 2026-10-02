import crypto from "node:crypto"

export function randomFrom(seed) {
    let state = crypto.createHash("sha256").update(seed).digest().readUInt32LE(0)

    return () => {
        state = (state + 0x6d2b79f5) >>> 0
        let t = state
        t = Math.imul(t ^ (t >>> 15), t | 1)
        t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296
    }
}

export function shuffled(items, seed) {
    const random = randomFrom(seed)
    const out = [...items]
    for (let i = out.length - 1; i > 0; i--) {
        const j = Math.floor(random() * (i + 1))
        ;[out[i], out[j]] = [out[j], out[i]]
    }

    return out
}

export function first(items, count) {
    const out = []
    const seen = new Set()
    for (const item of items) {
        if (out.length === count) break
        if (!item || seen.has(item)) continue
        seen.add(item)
        out.push(item)
    }

    return out
}

export function pick(items, count, seed) {
    return first(shuffled(items, seed), count)
}
