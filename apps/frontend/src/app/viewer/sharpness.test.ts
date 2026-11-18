import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {SHARP} from "./graphics.ts"
import {RATIO_STORAGE_KEY} from "./ratioStore.ts"
import {Sharpness} from "./sharpness.ts"

const AT_60HZ = 1_000 / 60

function storeOf(entries: Record<string, string> = {}) {
    const values = new Map(Object.entries(entries))
    return {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => void values.set(key, value),
        values,
    }
}

let frames: FrameRequestCallback[] = []
let at = 0

beforeEach(() => {
    frames = []
    at = performance.now()
    vi.stubGlobal("window", {devicePixelRatio: 3})
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => frames.push(callback))
})

afterEach(() => {
    vi.unstubAllGlobals()
})

// The browser's ticks while the board loads, before the loop draws.
function load(ticks: number) {
    for (let i = 0; i < ticks; i++) {
        at += AT_60HZ
        const due = frames
        frames = []
        for (const frame of due) frame(at)
    }
}

// The loop's ticks, each drawing a frame at 30 a second; how many times the ratio went down.
function drawSlowly(sharpness: Sharpness, ticks: number): number {
    let lowered = 0
    for (let i = 0; i < ticks; i++) {
        at += 2 * AT_60HZ
        sharpness.tick(at)
        if (sharpness.drew()) lowered++
    }
    return lowered
}

describe("Sharpness", () => {
    it("starts at the ratio a slow visit lowered it to", () => {
        const sharpness = new Sharpness(SHARP, true, new AbortController().signal, () => storeOf({[RATIO_STORAGE_KEY]: "1.5"}))

        expect(sharpness.ratio()).toBe(1.5)
    })

    it("goes from 2 to 1.5 to 1 on slow frames, and keeps it", () => {
        const store = storeOf()
        const sharpness = new Sharpness(SHARP, true, new AbortController().signal, () => store)
        load(60)
        sharpness.startMeasuring(at)

        expect(drawSlowly(sharpness, 200)).toBe(1)
        expect(sharpness.ratio()).toBe(1.5)
        expect(store.values.get(RATIO_STORAGE_KEY)).toBe("1.5")

        expect(drawSlowly(sharpness, 1_000)).toBe(1)
        expect(sharpness.ratio()).toBe(1)
        expect(store.values.get(RATIO_STORAGE_KEY)).toBe("1")
    })

    it("keeps the screen's ratio when the URL names it", () => {
        const store = storeOf({[RATIO_STORAGE_KEY]: "1"})
        const sharpness = new Sharpness({...SHARP, ratio: "screen"}, true, new AbortController().signal, () => store)
        load(60)
        sharpness.startMeasuring(at)

        expect(sharpness.ratio()).toBe(2)
        expect(drawSlowly(sharpness, 1_000)).toBe(0)
    })

    it("keeps the screen's ratio for a clip, which draws every frame", () => {
        const sharpness = new Sharpness(SHARP, false, new AbortController().signal, () => storeOf({[RATIO_STORAGE_KEY]: "1"}))
        load(60)
        sharpness.startMeasuring(at)

        expect(sharpness.ratio()).toBe(2)
        expect(drawSlowly(sharpness, 1_000)).toBe(0)
    })

    it("stays at 1 with HD graphics off", () => {
        const sharpness = new Sharpness({...SHARP, ratio: "one"}, true, new AbortController().signal, () => storeOf())

        expect(sharpness.ratio()).toBe(1)
    })

    it("stops the browser's ticks once the loop ticks", () => {
        const sharpness = new Sharpness(SHARP, true, new AbortController().signal, () => storeOf())
        load(5)

        sharpness.tick(at)
        load(1)

        expect(frames).toHaveLength(0)
    })

    it("stops the browser's ticks when the load is let go", () => {
        const loading = new AbortController()
        new Sharpness(SHARP, true, loading.signal, () => storeOf())
        load(5)

        loading.abort()
        load(1)

        expect(frames).toHaveLength(0)
    })
})
