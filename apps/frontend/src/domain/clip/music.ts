// Where a clip starts its anthem: the best highlight that leaves the anthem playing to the end of the clip, else early
// enough that it does, for a clip that ends in silence is worse than one that starts mid-phrase.
export function startOf(anthem: {seconds: number, at: readonly number[]} | undefined, clipSeconds: number): number {
    if (anthem === undefined || anthem.at.length === 0) return 0
    return anthem.at.find((at) => at + clipSeconds <= anthem.seconds)
        ?? Math.max(0, Math.min(...anthem.at, anthem.seconds - clipSeconds))
}
