import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {sharpOf} from "./graphics.ts"
import {Sharpness} from "./sharpness.ts"

const ADAPTIVE = sharpOf(true)

const AT_60HZ = 1_000 / 60

const OLD_KEY = "clickplanet-pixel-ratio"

function storeOf(entries: Record<string, string> = {}) {
    const values = new Map(Object.entries(entries))
    return {
        removeItem: (key: string) => void values.delete(key),
        values,
    }
}

let frames: FrameRequestCallback[] = []
let at = 0

beforeEach(() => {
    frames = []
    at = performance.now()
    vi.stubGlobal("window", {devicePixelRatio: 3, localStorage: storeOf()})
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => frames.push(callback))
})

afterEach(() => {
    vi.unstubAllGlobals()
})

function load(ticks: number) {
    for (let i = 0; i < ticks; i++) {
        at += AT_60HZ
        const due = frames
        frames = []
        for (const frame of due) frame(at)
    }
}

function measuredSharpness(graphics = ADAPTIVE, measured = true): Sharpness {
    const sharpness = new Sharpness(graphics, measured, new AbortController().signal)
    load(60)
    sharpness.startMeasuring(at)
    return sharpness
}

function drawFor(sharpness: Sharpness, seconds: number, frameOf: (ratio: number) => number): number[] {
    const until = at + seconds * 1_000
    const ratios: number[] = []
    while (at < until) {
        at += frameOf(sharpness.ratio())
        sharpness.tick(at)
        if (sharpness.drew()) ratios.push(sharpness.ratio())
    }
    return ratios
}

describe("Sharpness", () => {
    it("forgets the ratio an old visit kept, and starts at the screen's", () => {
        const store = storeOf({[OLD_KEY]: "1"})
        vi.stubGlobal("window", {devicePixelRatio: 3, localStorage: store})

        const sharpness = new Sharpness(ADAPTIVE, true, new AbortController().signal)

        expect(store.values.has(OLD_KEY)).toBe(false)
        expect(sharpness.ratio()).toBe(2)
    })

    it("starts when the storage is blocked", () => {
        vi.stubGlobal("window", {
            devicePixelRatio: 3,
            get localStorage(): Storage {
                throw new DOMException("blocked", "SecurityError")
            },
        })

        expect(new Sharpness(ADAPTIVE, true, new AbortController().signal).ratio()).toBe(2)
    })

    it("goes from 2 to 1.5 to 1 while the pixels make the frames slow", () => {
        const sharpness = measuredSharpness()

        expect(drawFor(sharpness, 60, (ratio) => 16 * ratio * ratio)).toEqual([1.5, 1])
    })

    it("puts the ratio back to 2 when a step does not make the frames faster", () => {
        const sharpness = measuredSharpness()

        expect(drawFor(sharpness, 60, () => 2 * AT_60HZ)).toEqual([1.5, 2])
        expect(drawFor(sharpness, 60, () => 2 * AT_60HZ)).toEqual([])
    })

    it("keeps the screen's ratio when the URL names it", () => {
        const sharpness = measuredSharpness({...ADAPTIVE, ratio: "screen"})

        expect(sharpness.ratio()).toBe(2)
        expect(drawFor(sharpness, 60, () => 2 * AT_60HZ)).toEqual([])
    })

    it("keeps the screen's ratio for a clip, which draws every frame", () => {
        const sharpness = measuredSharpness(ADAPTIVE, false)

        expect(sharpness.ratio()).toBe(2)
        expect(drawFor(sharpness, 60, () => 2 * AT_60HZ)).toEqual([])
    })

    it("stays at 1 with HD graphics off", () => {
        const sharpness = new Sharpness({...ADAPTIVE, ratio: "one"}, true, new AbortController().signal)

        expect(sharpness.ratio()).toBe(1)
    })

    it("stops the browser's ticks once the loop ticks", () => {
        const sharpness = new Sharpness(ADAPTIVE, true, new AbortController().signal)
        load(5)

        sharpness.tick(at)
        load(1)

        expect(frames).toHaveLength(0)
    })

    it("stops the browser's ticks when the load is let go", () => {
        const loading = new AbortController()
        new Sharpness(ADAPTIVE, true, loading.signal)
        load(5)

        loading.abort()
        load(1)

        expect(frames).toHaveLength(0)
    })
})
