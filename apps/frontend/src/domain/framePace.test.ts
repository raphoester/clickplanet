import {describe, expect, it} from "vitest"
import {FramePace, loweredRatio, paid, slow} from "./framePace.ts"

const AT_60HZ = 1_000 / 60
const AT_120HZ = 1_000 / 120
const AT_30HZ = 1_000 / 30

const times = (count: number, ms: number) => Array<number>(count).fill(ms)

const gpuBound = (atTwo: number) => (ratio: number) => (atTwo * ratio * ratio) / 4

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

describe("paid", () => {
    it("is not the same frame, as under a cap of the browser", () => {
        expect(paid(AT_30HZ, AT_30HZ)).toBe(false)
    })

    it("is not a frame a little shorter", () => {
        expect(paid(40, 36)).toBe(false)
    })

    it("is a frame clearly shorter", () => {
        expect(paid(40, 22.5)).toBe(true)
    })

    it("is one step of the display's, at 60 Hz and at 120 Hz", () => {
        expect(paid(2 * AT_60HZ, AT_60HZ)).toBe(true)
        expect(paid(4 * AT_120HZ, 3 * AT_120HZ)).toBe(true)
    })
})

type FrameMs = number | ((ratio: number) => number)

class Ticks {
    readonly pace = new FramePace(2)
    readonly changes: {at: number, cap: number}[] = []
    at = 1_000

    constructor(private readonly screen = 2) {}

    idle(count: number, ms: number) {
        for (let i = 0; i < count; i++) this.tick(ms, false)
    }

    draw(count: number, ms: FrameMs): number {
        let changed = 0
        for (let i = 0; i < count; i++) if (this.tick(this.frame(ms), true)) changed++
        return changed
    }

    drawFor(seconds: number, ms: FrameMs): number {
        const until = this.at + seconds * 1_000
        let changed = 0
        while (this.at < until) if (this.tick(this.frame(ms), true)) changed++
        return changed
    }

    tick(ms: number, drawn: boolean): boolean {
        this.at += ms
        this.pace.tick(this.at)
        if (!drawn || !this.pace.drew(this.ratio())) return false
        this.changes.push({at: this.at, cap: this.pace.cap()})
        return true
    }

    ratio(): number {
        return Math.min(this.screen, this.pace.cap())
    }

    measure() {
        this.pace.startMeasuring(this.at)
    }

    private frame(ms: FrameMs): number {
        return typeof ms === "number" ? ms : ms(this.ratio())
    }
}

function measured(cadence = AT_60HZ, screen = 2): Ticks {
    const ticks = new Ticks(screen)
    ticks.idle(5, cadence)
    ticks.measure()
    return ticks
}

const SETTLING_FRAMES = Math.ceil(3_000 / (2 * AT_60HZ))

describe("FramePace", () => {
    it("starts at the cap it is given", () => {
        expect(new FramePace(2).cap()).toBe(2)
    })

    it("lowers a 60 Hz screen that draws at 30 to 1.5", () => {
        const ticks = measured()

        expect(ticks.draw(SETTLING_FRAMES + 60, 2 * AT_60HZ)).toBe(1)
        expect(ticks.pace.cap()).toBe(1.5)
    })

    it("measures nothing until the tiles are read", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)

        expect(ticks.draw(1_000, 2 * AT_60HZ)).toBe(0)
    })

    it("leaves out the first three seconds after the tiles are read", () => {
        const ticks = measured()

        expect(ticks.draw(SETTLING_FRAMES + 58, 2 * AT_60HZ)).toBe(0)
    })

    it("does not lower a phone in Low Power Mode from the load, held at 30", () => {
        const ticks = measured(AT_30HZ)

        expect(ticks.draw(1_000, AT_30HZ)).toBe(0)
        expect(ticks.pace.cap()).toBe(2)
    })

    it("does not lower a screen at ratio 1", () => {
        const ticks = measured(AT_60HZ, 1)

        expect(ticks.draw(1_000, 2 * AT_60HZ)).toBe(0)
        expect(ticks.pace.cap()).toBe(2)
    })

    it("waits for the display's own interval", () => {
        const ticks = new Ticks()
        ticks.measure()

        expect(ticks.draw(1_000, 2 * AT_60HZ)).toBe(0)
    })

    it("does not measure a frame drawn after a tick that drew nothing", () => {
        const ticks = measured()

        let changed = 0
        for (let i = 0; i < 1_000; i++) {
            ticks.tick(AT_60HZ, false)
            if (ticks.tick(3 * AT_60HZ, true)) changed++
        }
        expect(changed).toBe(0)
    })

    it("does not judge the frames before a hidden tab with those after it", () => {
        const ticks = measured()
        ticks.draw(SETTLING_FRAMES + 59, 2 * AT_60HZ)

        ticks.tick(5_000, true)

        expect(ticks.draw(SETTLING_FRAMES, 2 * AT_60HZ)).toBe(0)
    })

    it("does not take a sleep of the loop for the display's own interval", () => {
        const ticks = new Ticks()
        ticks.idle(5, AT_60HZ)
        for (let i = 0; i < 10; i++) {
            ticks.tick(200, false)
            ticks.pace.pause()
        }
        ticks.measure()

        expect(ticks.draw(SETTLING_FRAMES + 60, 2 * AT_60HZ)).toBe(1)
    })

    it("takes the display's own interval only after a tick that drew nothing", () => {
        const ticks = new Ticks()
        ticks.measure()

        let changed = 0
        for (let i = 0; i < 60; i++) {
            changed += ticks.draw(10, 2 * AT_60HZ)
            ticks.tick(100, false)
            if (ticks.tick(AT_60HZ, true)) changed++
        }
        expect(changed).toBeGreaterThan(0)
    })
})

describe("FramePace, after a step", () => {
    it("judges the step on a new window, after the settle", () => {
        const ticks = measured()
        ticks.draw(SETTLING_FRAMES + 60, 2 * AT_60HZ)

        expect(ticks.draw(SETTLING_FRAMES, 2 * AT_60HZ)).toBe(0)
        expect(ticks.pace.cap()).toBe(1.5)
        expect(ticks.draw(61, 2 * AT_60HZ)).toBe(1)
        expect(ticks.pace.cap()).toBe(2)
    })

    it("puts back the step a 30 fps cap from mid-visit did not pay for, and lowers no more", () => {
        const ticks = measured()
        expect(ticks.drawFor(60, AT_60HZ)).toBe(0)

        const capped = ticks.at
        ticks.drawFor(120, AT_30HZ)

        expect(ticks.changes.map(({cap}) => cap)).toEqual([1.5, 2])
        const [lowered, undone] = ticks.changes.map(({at}) => at - capped)
        expect(lowered).toBeLessThan(1_500)
        expect(undone).toBeGreaterThan(5_000)
        expect(undone).toBeLessThan(7_000)
    })

    it("keeps a step the pixels paid for", () => {
        const ticks = measured()

        ticks.drawFor(60, gpuBound(40))

        expect(ticks.changes.map(({cap}) => cap)).toEqual([1.5])
    })

    it("goes on to 1 while the frames the pixels paid for are still slow", () => {
        const ticks = measured()

        ticks.drawFor(60, gpuBound(64))

        expect(ticks.changes.map(({cap}) => cap)).toEqual([1.5, 1])
    })

    it("puts back a step when the frames are as long at every ratio", () => {
        const ticks = measured()

        ticks.drawFor(60, 40)

        expect(ticks.changes.map(({cap}) => cap)).toEqual([1.5, 2])
    })

    it("puts back a step the pixels paid for at first, when the next one does not pay", () => {
        const ticks = measured()

        ticks.drawFor(60, (ratio) => (ratio === 2 ? 64 : 36))

        expect(ticks.changes.map(({cap}) => cap)).toEqual([1.5, 1, 1.5])
    })

    it("judges the step on the frames after a hidden tab", () => {
        const ticks = measured()
        ticks.draw(SETTLING_FRAMES + 60, 40)

        ticks.tick(5_000, true)
        ticks.draw(SETTLING_FRAMES + 60, 22.5)

        expect(ticks.changes.map(({cap}) => cap)).toEqual([1.5])
    })
})
