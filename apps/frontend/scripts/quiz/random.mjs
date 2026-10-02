// A draw that is the same every run.
//
// The bank is committed and content-addressed, so a generator that picked its distractors with
// `Math.random` would write a different file from the same data and make every regeneration look
// like a change. Everything below is seeded by the question's own id instead: rerunning is a no-op,
// and adding a country only moves the questions about that country.
import crypto from "node:crypto"

/** mulberry32, seeded from the sha256 of a string. */
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

/** A copy of `items`, shuffled the one way `seed` shuffles it. */
export function shuffled(items, seed) {
    const random = randomFrom(seed)
    const out = [...items]
    for (let i = out.length - 1; i > 0; i--) {
        const j = Math.floor(random() * (i + 1))
        ;[out[i], out[j]] = [out[j], out[i]]
    }

    return out
}

/**
 * The first `count` distinct items, in the order they came in. The templates hand this a list
 * already ranked by how good a wrong answer is, and a shuffle here would throw that away.
 * Duplicates and blanks are dropped: two countries can share the name of a capital.
 */
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

/** `count` of `items`, drawn the one way `seed` draws them. For lists with no order worth keeping. */
export function pick(items, count, seed) {
    return first(shuffled(items, seed), count)
}
