import {FramePace} from "../../domain/framePace.ts"
import {type Graphics} from "./graphics.ts"
import {MAX_PIXEL_RATIO, pixelRatio} from "./scene.ts"

const PROBE_MS = 10_000

const OLD_RATIO_STORAGE_KEY = "clickplanet-pixel-ratio"

export class Sharpness {
    private readonly pace = new FramePace(MAX_PIXEL_RATIO)
    private readonly adapts: boolean
    private looping = false

    constructor(
        private readonly graphics: Graphics,
        measured: boolean,
        signal: AbortSignal,
    ) {
        forgetOldRatio()
        this.adapts = measured && graphics.ratio === "adaptive"
        if (this.adapts) this.probe(signal)
    }

    ratio(): number {
        return pixelRatio(this.graphics, this.pace.cap())
    }

    tick(at: number): void {
        this.looping = true
        if (this.adapts) this.pace.tick(at)
    }

    pause(): void {
        if (this.adapts) this.pace.pause()
    }

    drew(): boolean {
        return this.adapts && this.pace.drew(this.ratio())
    }

    startMeasuring(at: number): void {
        this.pace.startMeasuring(at)
    }

    private probe(signal: AbortSignal) {
        const until = performance.now() + PROBE_MS
        const step = (at: number) => {
            if (this.looping || signal.aborted || at > until) return
            this.pace.tick(at)
            requestAnimationFrame(step)
        }
        requestAnimationFrame(step)
    }
}

function forgetOldRatio() {
    try {
        window.localStorage.removeItem(OLD_RATIO_STORAGE_KEY)
    } catch {
        // No storage: nothing was kept.
    }
}
