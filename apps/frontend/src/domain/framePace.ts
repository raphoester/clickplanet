const FRAMES = 60

const CADENCE_TICKS = 5

const GAP_MS = 1_000

const SETTLE_MS = 3_000

const SLOW = 1.75

const AIMED_MS = 1_000 / 60

const RATIO_STEPS = [1.5, 1]

export function loweredRatio(ratio: number): number {
    return RATIO_STEPS.find((step) => step < ratio) ?? ratio
}

export type Pace = {
    // The intervals between two ticks that both drew a frame.
    frames: readonly number[]
    // The intervals after a tick that drew nothing: the display's own.
    cadence: readonly number[]
}

export function slow({frames, cadence}: Pace): boolean {
    if (frames.length < FRAMES || cadence.length < CADENCE_TICKS) return false
    return median(frames) > SLOW * Math.max(median(cadence), AIMED_MS)
}

export class FramePace {
    private readonly frames: number[] = []
    private readonly cadence: number[] = []
    private tickedAt: number | undefined
    private sinceLastTick = 0
    private drawnBefore = false
    private drawn = false
    private measuresFrom = Infinity

    tick(at: number): void {
        this.sinceLastTick = this.tickedAt === undefined ? 0 : at - this.tickedAt
        this.tickedAt = at
        this.drawnBefore = this.drawn
        this.drawn = false
        if (this.sinceLastTick > GAP_MS) this.settle(at)
        else if (!this.drawnBefore && this.sinceLastTick > 0) keep(this.cadence, this.sinceLastTick, CADENCE_TICKS)
    }

    // True when the frames have come slow; they are then measured again from scratch.
    drew(): boolean {
        this.drawn = true
        if (!this.drawnBefore || this.tickedAt === undefined || this.tickedAt < this.measuresFrom) return false
        keep(this.frames, this.sinceLastTick, FRAMES)
        if (!slow({frames: this.frames, cadence: this.cadence})) return false
        this.settle(this.tickedAt)
        return true
    }

    startMeasuring(at: number): void {
        this.frames.length = 0
        this.measuresFrom = at + SETTLE_MS
    }

    private settle(at: number) {
        this.frames.length = 0
        this.measuresFrom = Math.max(this.measuresFrom, at + SETTLE_MS)
    }
}

function keep(values: number[], value: number, most: number) {
    values.push(value)
    if (values.length > most) values.shift()
}

function median(values: readonly number[]): number {
    const sorted = [...values].sort((a, b) => a - b)
    const middle = sorted.length / 2
    return Number.isInteger(middle) ? (sorted[middle - 1]! + sorted[middle]!) / 2 : sorted[Math.floor(middle)]!
}
