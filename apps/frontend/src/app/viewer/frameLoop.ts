// What a sleeping loop reads without being told (a box offered, a rest on a tile) is late by this much at most.
export const IDLE_TICK_MS = 200

export type FrameSource = {
    requestFrame: (tick: (time: number) => void) => number
    cancelFrame: (id: number) => void
    setTimer: (run: () => void, ms: number) => ReturnType<typeof setTimeout>
    clearTimer: (id: ReturnType<typeof setTimeout>) => void
}

// A tick answers whether the next display frame is wanted.
export type Tick = (time: number) => boolean

// A display frame asked for wakes the main thread, drawn or not: with no work, the next tick is IDLE_TICK_MS away.
export class FrameLoop {
    private tick: Tick | undefined
    private frame: number | undefined
    private timer: ReturnType<typeof setTimeout> | undefined

    constructor(private readonly source: FrameSource = browserFrames) {
    }

    start(tick: Tick): void {
        this.tick = tick
        this.wake()
    }

    wake(): void {
        if (!this.tick || this.frame !== undefined) return
        if (this.timer !== undefined) this.source.clearTimer(this.timer)
        this.timer = undefined
        this.frame = this.source.requestFrame(this.run)
    }

    stop(): void {
        this.tick = undefined
        if (this.frame !== undefined) this.source.cancelFrame(this.frame)
        if (this.timer !== undefined) this.source.clearTimer(this.timer)
        this.frame = undefined
        this.timer = undefined
    }

    private readonly run = (time: number) => {
        this.frame = undefined
        const tick = this.tick
        if (!tick) return
        const busy = tick(time)
        if (!this.tick || this.frame !== undefined) return
        if (busy) {
            this.frame = this.source.requestFrame(this.run)
            return
        }
        this.timer = this.source.setTimer(() => {
            this.timer = undefined
            this.wake()
        }, IDLE_TICK_MS)
    }
}

const browserFrames: FrameSource = {
    requestFrame: (tick) => requestAnimationFrame(tick),
    cancelFrame: (id) => cancelAnimationFrame(id),
    setTimer: (run, ms) => setTimeout(run, ms),
    clearTimer: (id) => clearTimeout(id),
}
