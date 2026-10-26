export type SlotCandidate = {
    landmass: number
    screenWidth: number
    spriteWidth: number
    inView: boolean
}

export const MAGNIFIED_FROM = 1.5

export const SETTLE_SECONDS = 0.15

// The landmass whose painted flag is widest on screen, among those drawn wider than their sprite can show.
export function magnifiedMost(candidates: Iterable<SlotCandidate>, from = MAGNIFIED_FROM): number | undefined {
    let best: SlotCandidate | undefined
    for (const candidate of candidates) {
        if (!candidate.inView || candidate.screenWidth < candidate.spriteWidth * from) continue
        if (best === undefined || candidate.screenWidth > best.screenWidth) best = candidate
    }
    return best?.landmass
}

// Holds the choice steady through a pan: it changes only once the camera has been still for a while.
export class SlotChoice {
    private turnedAt = -Infinity
    private chosen: number | undefined

    constructor(private readonly settleSeconds = SETTLE_SECONDS) {}

    choose(turned: boolean, seconds: number, bestOf: () => number | undefined): number | undefined {
        if (turned) this.turnedAt = seconds
        if (seconds - this.turnedAt < this.settleSeconds) return this.chosen
        this.chosen = bestOf()
        return this.chosen
    }
}
