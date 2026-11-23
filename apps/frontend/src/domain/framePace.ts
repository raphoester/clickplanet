const FRAMES = 60

const CADENCE_TICKS = 5

const GAP_MS = 1_000

const SETTLE_MS = 3_000

const SLOW = 1.75

const PAID = 0.85

const AIMED_MS = 1_000 / 60

const RATIO_STEPS = [1.5, 1]

export function loweredRatio(ratio: number): number {
    return RATIO_STEPS.find((step) => step < ratio) ?? ratio
}

export type Pace = {
    frames: readonly number[]
    cadence: readonly number[]
}

export function slow({frames, cadence}: Pace): boolean {
    if (frames.length < FRAMES || cadence.length < CADENCE_TICKS) return false
    return median(frames) > SLOW * Math.max(median(cadence), AIMED_MS)
}

export function paid(before: number, after: number): boolean {
    return after <= PAID * before
}

type Step = {
    from: number
    median: number
}

export class FramePace {
    private readonly frames: number[] = []
    private readonly cadence: number[] = []
    private tickedAt: number | undefined
    private sinceLastTick = 0
    private drawnBefore = false
    private drawn = false
    private measuresFrom = Infinity
    private step: Step | undefined
    private held = false

    constructor(private most: number) {}

    cap(): number {
        return this.most
    }

    tick(at: number): void {
        this.sinceLastTick = this.tickedAt === undefined ? 0 : at - this.tickedAt
        this.tickedAt = at
        this.drawnBefore = this.drawn
        this.drawn = false
        if (this.sinceLastTick > GAP_MS) this.settle(at)
        else if (!this.drawnBefore && this.sinceLastTick > 0) keep(this.cadence, this.sinceLastTick, CADENCE_TICKS)
    }

    drew(ratio: number): boolean {
        this.drawn = true
        if (this.held || !this.drawnBefore || this.tickedAt === undefined || this.tickedAt < this.measuresFrom) return false
        keep(this.frames, this.sinceLastTick, FRAMES)
        if (this.frames.length < FRAMES) return false
        const step = this.step
        this.step = undefined
        if (step !== undefined && !paid(step.median, median(this.frames))) {
            this.most = step.from
            this.held = true
            return true
        }
        if (!slow({frames: this.frames, cadence: this.cadence})) return false
        const lowered = loweredRatio(ratio)
        if (lowered === ratio) return false
        this.step = {from: this.most, median: median(this.frames)}
        this.most = lowered
        this.settle(this.tickedAt)
        return true
    }

    pause(): void {
        this.tickedAt = undefined
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
