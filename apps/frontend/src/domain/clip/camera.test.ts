import {describe, expect, it} from "vitest"
import {blastZoomOf, cameraOf, framingOf, openingOf} from "./camera.ts"
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
    const beats = Array.from({length: 100}, (_, i) => ({share: i / 100, point: at(0.3)}))
    const seconds = 10
    const camera = cameraOf(opening, beats, {close: 6, seconds, pullBack: "atEnd" as const})
    const after = (time: number) => camera(time / seconds)

    it("holds on the opening shot for a moment", () => {
        expect(after(0).zoom).toBeCloseTo(1.5)
        expect(after(0.25).zoom).toBeCloseTo(1.5)
    })

    it("then dives fast into where the tiles change hands", () => {
        expect(after(0.65).zoom).toBeGreaterThan(2.5)
        expect(after(1.1).zoom).toBeCloseTo(6)
        expect(after(5).direction.x).toBeCloseTo(Math.sin(0.3))
    })

    it("holds as long as it is told before it dives", () => {
        const late = cameraOf(opening, beats, {close: 6, seconds, pullBack: "atEnd" as const, hold: 1.5})

        expect(late(1.4 / seconds).zoom).toBeCloseTo(1.5)
        expect(late(2.3 / seconds).zoom).toBeCloseTo(6)
    })

    it("pulls back out to the opening at the end", () => {
        expect(after(10).zoom).toBeCloseTo(1.5)
        expect(after(10).direction.x).toBeLessThan(0.1)
    })

    it("stays in the tiles to the end when it never pulls back", () => {
        const kept = cameraOf(opening, beats, {close: 6, seconds, pullBack: "never"})

        expect(kept(1).zoom).toBeCloseTo(6)
    })

    it("pulls back halfway and stays out, for a steamroll's painted flags", () => {
        const steamroll = cameraOf(opening, beats, {close: 6, seconds, pullBack: "midway"})

        expect(steamroll(0.3).zoom).toBeCloseTo(6)
        expect(steamroll(0.75).zoom).toBeCloseTo(1.5)
        expect(steamroll(1).zoom).toBeCloseTo(1.5)
    })

    it("stays on the opening's direction where nothing happens", () => {
        const {direction} = cameraOf(opening, [], {close: 6, seconds, pullBack: "atEnd" as const})(0.5)

        expect(direction.x).toBeCloseTo(0)
        expect(direction.z).toBeCloseTo(1)
    })
})

describe("a blast", () => {
    const opening = {direction: at(0), zoom: 1.5}
    const blast = {from: 0.4, to: 0.5, point: at(-0.2), zoom: 7}
    const script = {close: 3, seconds: 10, pullBack: "atEnd" as const, blasts: [blast]}

    it("takes the camera to it, close, while the clip holds on it", () => {
        const shot = cameraOf(opening, [], script)(0.45)

        expect(shot.direction.x).toBeCloseTo(Math.sin(-0.2), 2)
        expect(shot.zoom).toBeGreaterThan(6)
    })

    it("lets it go after", () => {
        expect(cameraOf(opening, [], script)(0.7).zoom).toBeCloseTo(3)
    })

    it("still pulls back out at the end when it falls late", () => {
        const late = {...blast, from: 0.88, to: 0.9}
        const camera = cameraOf(opening, [], {...script, blasts: [late]})

        expect(camera(0.89).zoom).toBeGreaterThan(6)
        expect(camera(1).zoom).toBeCloseTo(1.5)
        expect(camera(1).direction.x).toBeCloseTo(0)
    })

    it("is a fifth of the screen wide", () => {
        expect(blastZoomOf(0.016, 0.5625)).toBeCloseTo(0.5625 / 0.08)
        expect(blastZoomOf(0.001, 0.5625)).toBe(9)
    })
})
