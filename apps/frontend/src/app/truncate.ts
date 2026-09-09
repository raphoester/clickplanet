export function truncate(text: string, max: number): string {
    const units = segment(text)
    if (units.length <= max) return text

    return units.slice(0, max).join("").trimEnd() + "…"
}

const segmenter = typeof Intl !== "undefined" && "Segmenter" in Intl
    ? new Intl.Segmenter(undefined, {granularity: "grapheme"})
    : undefined

function segment(text: string): string[] {
    if (!segmenter) return Array.from(text)
    return Array.from(segmenter.segment(text), s => s.segment)
}
