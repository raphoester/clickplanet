import {describe, expect, it} from "vitest"
import {FramePace, loweredRatio, slow} from "./framePace.ts"

const AT_60HZ = 1_000 / 60
const AT_120HZ = 1_000 / 120
const AT_30HZ = 1_000 / 30

const times = (count: number, ms: number) => Array<number>(count).fill(ms)

describe("loweredRatio", () => {
    it("steps from 2 to 1.5 to 1", () => {
        expect(loweredRatio(2)).toBe(1.5)
        expect(loweredRatio(1.5)).toBe(1)
    })

    it("takes a screen between two steps to the step below it", () => {
        expect(loweredRatio(1.75)).toBe(1.5)
        expect(loweredRatio(1.25)).toBe(1)
    })

    it("goes no lower than 1", () => {
        expect(loweredRatio(1)).toBe(1)
    })
})

describe("slow", () => {
    it("is a 60 Hz screen at 30 frames a second", () => {
        expect(slow({frames: times(60, 2 * AT_60HZ), cadence: times(5, AT_60HZ)})).toBe(true)
    })

    it("is not a 60 Hz screen at 60", () => {
        expect(slow({frames: times(60, AT_60HZ), cadence: times(5, AT_60HZ)})).toBe(false)
    })

    it("is not a 120 Hz screen at 60, nor at 40", () => {
        expect(slow({frames: times(60, 2 * AT_120HZ), cadence: times(5, AT_120HZ)})).toBe(false)
        expect(slow({frames: times(60, 3 * AT_120HZ), cadence: times(5, AT_120HZ)})).toBe(false)
    })

    it("is a 120 Hz screen at 30", () => {
        expect(slow({frames: times(60, 4 * AT_120HZ), cadence: times(5, AT_120HZ)})).toBe(true)
    })

    it("is not a tab the browser holds at 30 frames a second, until it draws at 15", () => {
        expect(slow({frames: times(60, AT_30HZ), cadence: times(5, AT_30HZ)})).toBe(false)
        expect(slow({frames: times(60, 2 * AT_30HZ), cadence: times(5, AT_30HZ)})).toBe(true)
    })

    it("is not one hitch", () => {
        expect(slow({frames: [...times(59, AT_60HZ), 900], cadence: times(5, AT_60HZ)})).toBe(false)
    })

    it("is when more than half the frames miss the display's", () => {
        const cadence = times(5, AT_60HZ)
        expect(slow({frames: [...times(31, 2 * AT_60HZ), ...times(29, AT_60HZ)], cadence})).toBe(true)
        expect(slow({frames: [...times(30, 2 * AT_60HZ), ...times(30, AT_60HZ)], cadence})).toBe(false)
    })

    it("waits for a full window of frames", () => {
        expect(slow({frames: times(59, 2 * AT_60HZ), cadence: times(5, AT_60HZ)})).toBe(false)
    })

    it("waits until the display's own interval is known", () => {
        expect(slow({frames: times(60, 2 * AT_60HZ), cadence: times(4, AT_60HZ)})).toBe(false)
    })
})

class Ticks {
    readonly pace = new FramePace()
    at = 1_000

    // Ticks that draw nothing: the display's own pace, before the loop draws.
    idle(count: number, ms: number) {
        for (let i = 0; i < count; i++) this.tick(ms, false)
    }

    // Ticks that each draw a frame; how many said the frames came slow.
    draw(count: number, ms: number): number {
        let slowed = 0
        for (let i = 0; i < count; i++) if (this.tick(ms, true)) slowed++
        return slowed
    }

    tick(ms: number, drawn: boolean): boolean {
        this.at += ms
        this.pace.tick(this.at)
        return drawn && this.pace.drew()
    }

    measure() {
        this.pace.startMeasuring(this.at)
    }
}

// The settling after the tiles are read, then a window of frames.
const SETTLING_FRAMES = Math.ceil(3_000 / (2 * AT_60HZ))

describe("FramePace", () => {
    it("says once that a 60 Hz screen draws at 30", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)
        ticks.measure()

        expect(ticks.draw(SETTLING_FRAMES + 60, 2 * AT_60HZ)).toBe(1)
    })

    it("judges again from scratch after it says so", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)
        ticks.measure()
        ticks.draw(SETTLING_FRAMES + 60, 2 * AT_60HZ)

        expect(ticks.draw(SETTLING_FRAMES, 2 * AT_60HZ)).toBe(0)
        expect(ticks.draw(61, 2 * AT_60HZ)).toBe(1)
    })

    it("measures nothing until the tiles are read", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)

        expect(ticks.draw(1_000, 2 * AT_60HZ)).toBe(0)
    })

    it("leaves out the first three seconds after the tiles are read", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)
        ticks.measure()

        expect(ticks.draw(SETTLING_FRAMES + 58, 2 * AT_60HZ)).toBe(0)
    })

    it("does not say so of a tab the browser holds at 30", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_30HZ)
        ticks.measure()

        expect(ticks.draw(1_000, AT_30HZ)).toBe(0)
    })

    it("waits for the display's own interval", () => {
        const ticks = new Ticks()
        ticks.measure()

        expect(ticks.draw(1_000, 2 * AT_60HZ)).toBe(0)
    })

    it("does not measure a frame drawn after a tick that drew nothing", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)
        ticks.measure()

        let slowed = 0
        for (let i = 0; i < 1_000; i++) {
            ticks.tick(AT_60HZ, false)
            if (ticks.tick(3 * AT_60HZ, true)) slowed++
        }
        expect(slowed).toBe(0)
    })

    it("does not judge the frames before a hidden tab with those after it", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)
        ticks.measure()
        ticks.draw(SETTLING_FRAMES + 59, 2 * AT_60HZ)

        ticks.tick(5_000, true)

        expect(ticks.draw(SETTLING_FRAMES, 2 * AT_60HZ)).toBe(0)
    })

    it("takes the display's own interval only after a tick that drew nothing", () => {
        const ticks = new Ticks()
        ticks.measure()

        let slowed = 0
        for (let i = 0; i < 60; i++) {
            slowed += ticks.draw(10, 2 * AT_60HZ)
            ticks.tick(100, false)
            if (ticks.tick(AT_60HZ, true)) slowed++
        }
        expect(slowed).toBeGreaterThan(0)
    })
})
