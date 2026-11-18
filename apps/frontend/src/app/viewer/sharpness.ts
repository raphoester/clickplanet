import {FramePace, loweredRatio} from "../../domain/framePace.ts"
import {type Graphics} from "./graphics.ts"
import {browserStore, type RatioStore, saveRatio, savedRatio} from "./ratioStore.ts"
import {MAX_PIXEL_RATIO, pixelRatio} from "./scene.ts"

// A globe whose load failed is never disposed: the probe stops on its own.
const PROBE_MS = 10_000

export class Sharpness {
    private readonly pace = new FramePace()
    private readonly adapts: boolean
    private most: number
    private looping = false

    constructor(
        private readonly graphics: Graphics,
        measured: boolean,
        signal: AbortSignal,
        private readonly storeOf: () => RatioStore = browserStore,
    ) {
        this.adapts = measured && graphics.ratio === "adaptive"
        this.most = this.adapts ? savedRatio(storeOf) : MAX_PIXEL_RATIO
        if (this.adapts) this.probe(signal)
    }

    ratio(): number {
        return pixelRatio(this.graphics, this.most)
    }

    tick(at: number): void {
        this.looping = true
        if (this.adapts) this.pace.tick(at)
    }

    // True when the frames came slow and the ratio went down: the canvas is to be sized again.
    drew(): boolean {
        if (!this.adapts || !this.pace.drew()) return false
        const ratio = this.ratio()
        const lowered = loweredRatio(ratio)
        if (lowered === ratio) return false
        this.most = lowered
        saveRatio(lowered, this.storeOf)
        return true
    }

    startMeasuring(at: number): void {
        this.pace.startMeasuring(at)
    }

    // Until the loop draws, every tick of the browser's comes at the display's own pace.
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
