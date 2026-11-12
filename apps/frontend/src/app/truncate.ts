import {translator} from "../i18n/page.ts"

export function truncate(text: string, max: number): string {
    const units = segment(text)
    if (units.length <= max) return text

    return units.slice(0, max).join("").trimEnd() + "…"
}

let segmenter: {locale: string, graphemes: Intl.Segmenter} | undefined

function segment(text: string): string[] {
    if (typeof Intl === "undefined" || !("Segmenter" in Intl)) return Array.from(text)
    const locale = translator().locale
    if (segmenter?.locale !== locale) segmenter = {locale, graphemes: new Intl.Segmenter(locale, {granularity: "grapheme"})}
    return Array.from(segmenter.graphemes.segment(text), s => s.segment)
}
