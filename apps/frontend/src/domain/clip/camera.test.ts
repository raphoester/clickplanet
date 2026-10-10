import {describe, expect, it} from "vitest"
import {cameraOf, framingOf, openingOf, screensOf} from "./camera.ts"
import {Point} from "./geometry.ts"

const at = (angle: number): Point => ({x: Math.sin(angle), y: 0, z: Math.cos(angle)})

describe("the framing of a front", () => {
    it("looks at the middle of it", () => {
        const {direction} = framingOf([at(-0.1), at(0.1)], 0.5625)

        expect(direction.x).toBeCloseTo(0)
        expect(direction.z).toBeCloseTo(1)
    })

    it("zooms in on a small front and out on a wide one", () => {
        expect(framingOf([at(-0.05), at(0.05)], 0.5625).zoom).toBeGreaterThan(framingOf([at(-0.3), at(0.3)], 0.5625).zoom)
    })

    it("never shows more than a little sky around the globe", () => {
        expect(framingOf([], 0.5625).zoom).toBeCloseTo(0.675)
    })
})

describe("the opening shot", () => {
    it("is the whole front, and never closer than a continent", () => {
        expect(openingOf({direction: at(0), zoom: 1.2}).zoom).toBe(1.2)
        expect(openingOf({direction: at(0), zoom: 4.6}).zoom).toBe(1.5)
    })
})

describe("the camera over a clip", () => {
    const opening = {direction: at(0), zoom: 1.5}
    const still = Array.from({length: 100}, (_, i) => ({share: i / 100, point: at(0.3)}))
    const moving = Array.from({length: 100}, (_, i) => ({share: i / 100, point: at(-1.2 + 2.4 * i / 100)}))
    const seconds = 10
    const camera = cameraOf(opening, still, {close: 6, seconds, pullBack: "atEnd" as const})
    const after = (time: number) => camera(time / seconds)

    it("holds on the opening shot for a moment", () => {
        expect(after(0).zoom).toBeCloseTo(1.5)
        expect(after(0.25).zoom).toBeCloseTo(1.5)
    })

    it("then dives fast into where the tiles change hands", () => {
        expect(after(0.65).zoom).toBeGreaterThan(2.5)
        expect(after(1).zoom).toBeCloseTo(6)
        expect(after(4).direction.x).toBeCloseTo(Math.sin(0.3), 1)
    })

    it("holds as long as it is told before it dives", () => {
        const late = cameraOf(opening, still, {close: 6, seconds, pullBack: "atEnd" as const, hold: 1.5})

        expect(late(1.4 / seconds).zoom).toBeCloseTo(1.5)
        expect(late(2.2 / seconds).zoom).toBeCloseTo(6)
    })

    it("breathes out to the painted flags and back in where the action stays in one place", () => {
        expect(after(2.5).zoom).toBeCloseTo(3, 1)
        expect(after(4).zoom).toBeCloseTo(6, 1)
        expect(after(5.5).zoom).toBeCloseTo(3, 1)
    })

    it("does not breathe where the action crosses the map", () => {
        const travelling = cameraOf(opening, moving, {close: 6, seconds, pullBack: "atEnd" as const})

        expect(travelling(0.25).zoom).toBeCloseTo(6, 1)
        expect(travelling(0.55).zoom).toBeCloseTo(6, 1)
    })

    it("pulls back out to the opening at the end, and holds there a second before the call to act", () => {
        expect(after(9).zoom).toBeCloseTo(1.5)
        expect(after(10).zoom).toBeCloseTo(1.5)
        expect(after(10).direction.x).toBeLessThan(0.1)
    })

    it("pulls back out at the end even when the action crosses the map", () => {
        expect(cameraOf(opening, moving, {close: 6, seconds, pullBack: "atEnd" as const})(0.9).zoom).toBeCloseTo(1.5)
    })

    it("pulls back halfway and stays out, for a steamroll's painted flags", () => {
        const steamroll = cameraOf(opening, moving, {close: 6, seconds, pullBack: "midway"})

        expect(steamroll(0.3).zoom).toBeCloseTo(6, 1)
        expect(steamroll(0.75).zoom).toBeCloseTo(1.5)
        expect(steamroll(1).zoom).toBeCloseTo(1.5)
    })

    it("stays on the opening's direction where nothing happens", () => {
        const {direction} = cameraOf(opening, [], {close: 6, seconds, pullBack: "atEnd" as const})(0.5)

        expect(direction.x).toBeCloseTo(0)
        expect(direction.z).toBeCloseTo(1)
    })
})

describe("how far the action moves", () => {
    const beats = (points: (i: number) => Point) => Array.from({length: 100}, (_, i) => ({share: i / 100, point: points(i)}))

    it("is nothing for a fight in one place", () => {
        expect(screensOf(beats(() => at(0.3)), 6)).toBeCloseTo(0)
    })

    it("is counted in screens up close", () => {
        expect(screensOf(beats((i) => at(-0.5 + i / 100)), 6)).toBeCloseTo(0.95 * 3, 0)
    })
})
