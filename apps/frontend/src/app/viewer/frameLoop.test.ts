import {describe, expect, it} from "vitest"
import {FrameLoop, FrameSource, IDLE_TICK_MS} from "./frameLoop.ts"

function fakeFrames() {
    let now = 0
    let ids = 0
    const frames = new Map<number, (time: number) => void>()
    const timers = new Map<number, {at: number, run: () => void}>()
    const source: FrameSource = {
        requestFrame: (tick) => {
            frames.set(++ids, tick)
            return ids
        },
        cancelFrame: (id) => {
            frames.delete(id)
        },
        setTimer: (run, ms) => {
            timers.set(++ids, {at: now + ms, run})
            return ids as unknown as ReturnType<typeof setTimeout>
        },
        clearTimer: (id) => {
            timers.delete(id as unknown as number)
        },
    }
    // One display frame: it runs the frames asked for before it; a frame a timer asks for is the next one.
    const frame = (ms = 16) => {
        now += ms
        const due = [...frames]
        frames.clear()
        for (const [id, timer] of [...timers]) {
            if (timer.at > now) continue
            timers.delete(id)
            timer.run()
        }
        for (const [, tick] of due) tick(now)
    }
    return {source, frame, pending: () => ({frames: frames.size, timers: timers.size})}
}

describe("FrameLoop", () => {
    it("ticks on every display frame while there is work", () => {
        const {source, frame} = fakeFrames()
        const ticks: number[] = []
        new FrameLoop(source).start((time) => {
            ticks.push(time)
            return true
        })

        frame()
        frame()
        frame()
        expect(ticks).toEqual([16, 32, 48])
    })

    it("ticks every IDLE_TICK_MS once there is none", () => {
        const {source, frame} = fakeFrames()
        const ticks: number[] = []
        new FrameLoop(source).start((time) => {
            ticks.push(time)
            return false
        })

        for (let i = 0; i < 45; i++) frame(10)
        expect(ticks).toEqual([10, 220, 430])
        expect(IDLE_TICK_MS).toBe(200)
    })

    it("comes back on the next frame when woken", () => {
        const {source, frame} = fakeFrames()
        const ticks: number[] = []
        const loop = new FrameLoop(source)
        loop.start((time) => {
            ticks.push(time)
            return false
        })
        frame(10)

        frame(10)
        loop.wake()
        frame(10)
        expect(ticks).toEqual([10, 30])
    })

    it("asks for one frame however many times it is woken", () => {
        const {source, frame, pending} = fakeFrames()
        let ticks = 0
        const loop = new FrameLoop(source)
        loop.start(() => {
            ticks++
            return true
        })

        loop.wake()
        loop.wake()
        expect(pending()).toEqual({frames: 1, timers: 0})
        frame()
        expect(ticks).toBe(1)
    })

    it("is woken from inside its own tick without a second frame", () => {
        const {source, frame, pending} = fakeFrames()
        const loop = new FrameLoop(source)
        loop.start(() => {
            loop.wake()
            return false
        })

        frame()
        expect(pending()).toEqual({frames: 1, timers: 0})
    })

    it("ticks no more once stopped, from outside or from its tick", () => {
        const {source, frame, pending} = fakeFrames()
        let ticks = 0
        const loop = new FrameLoop(source)
        loop.start(() => {
            ticks++
            if (ticks === 2) loop.stop()
            return true
        })

        frame()
        frame()
        frame()
        expect(ticks).toBe(2)
        expect(pending()).toEqual({frames: 0, timers: 0})

        loop.wake()
        expect(pending()).toEqual({frames: 0, timers: 0})
    })

    it("does nothing when woken before it is started", () => {
        const {source, pending} = fakeFrames()
        new FrameLoop(source).wake()
        expect(pending()).toEqual({frames: 0, timers: 0})
    })
})
