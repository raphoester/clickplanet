/**
 * Shortens a string to at most `max` user-perceived characters.
 *
 * Counting with `String.length` counts UTF-16 units, which is why the
 * leaderboard used to render England as "Eng": every country name is prefixed
 * with a flag, and the UK nations' flags are tag sequences fourteen units long,
 * so the emoji alone ate most of an eighteen-character budget. Regional
 * indicator pairs cost four. `Intl.Segmenter` counts each of them as one.
 */
export function truncate(text: string, max: number): string {
    const units = segment(text)
    if (units.length <= max) return text

    return units.slice(0, max).join("").trimEnd() + "…"
}

const segmenter = typeof Intl !== "undefined" && "Segmenter" in Intl
    ? new Intl.Segmenter(undefined, {granularity: "grapheme"})
    : undefined

function segment(text: string): string[] {
    /** Code points are the fallback: still never splits a surrogate pair. */
    if (!segmenter) return Array.from(text)
    return Array.from(segmenter.segment(text), s => s.segment)
}
