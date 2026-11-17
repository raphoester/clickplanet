export type Span = {start: number, count: number}

// `sorted` ascending; each value covers `width` items.
export function spansOf(sorted: ArrayLike<number>, width = 1): Span[] {
    const spans: Span[] = []
    let last: Span | undefined
    for (let i = 0; i < sorted.length; i++) {
        const start = sorted[i]
        if (last && start <= last.start + last.count) {
            last.count = Math.max(last.count, start + width - last.start)
            continue
        }
        last = {start, count: width}
        spans.push(last)
    }
    return spans
}

// Joins every gap under `least`, then the shortest others until at most `most` spans are left.
export function joinSpans(spans: readonly Span[], most: number, least = 0): Span[] {
    if (spans.length === 0) return []

    const gapOf = (i: number) => spans[i + 1].start - (spans[i].start + spans[i].count)
    // The most - 1 widest gaps stay open, widest first; a tie keeps the earlier gap.
    const kept = Math.max(most, 1) - 1
    const widest: number[] = []
    for (let i = 0; i < spans.length - 1; i++) {
        const gap = gapOf(i)
        if (gap < least) continue
        if (widest.length === kept && (kept === 0 || gap <= gapOf(widest[kept - 1]))) continue
        let at = widest.length === kept ? kept - 1 : widest.length
        while (at > 0 && gapOf(widest[at - 1]) < gap) {
            widest[at] = widest[at - 1]
            at--
        }
        widest[at] = i
    }
    const open = new Set(widest)

    const out: Span[] = []
    let current = {start: spans[0].start, count: spans[0].count}
    for (let i = 0; i < spans.length - 1; i++) {
        const next = spans[i + 1]
        if (open.has(i)) {
            out.push(current)
            current = {start: next.start, count: next.count}
        } else {
            current.count = next.start + next.count - current.start
        }
    }
    out.push(current)
    return out
}
