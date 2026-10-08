import {describe, expect, it} from "vitest"
import {widestZoomOf} from "./camera.ts"
import {Point} from "./geometry.ts"
import {stopsOf, tourOf, tourSecondsOf, Visit} from "./tour.ts"

const at = (angle: number): Point => ({x: Math.sin(angle), y: 0, z: Math.cos(angle)})
const ASPECT = 0.5625

function visits(region: string | undefined, count: number, angle: number, from: number, to: number): Visit[] {
    return Array.from({length: count}, (_, i) => ({region, point: at(angle + 0.01 * (i % 5)), share: from + (to - from) * i / count}))
}

describe("the stops of a tour", () => {
    const portugal = [
        ...visits("Asia", 40, 1.2, 0.5, 1), ...visits("Europe", 40, 0, 0, 1), ...visits("Africa", 20, 0.3, 0.2, 0.6),
        ...visits("Oceania", 3, 2.5, 0.1, 0.2), ...visits(undefined, 10, 0, 0, 1),
    ]

    it("are the continents the flag took land on, in the order it got there", () => {
        expect(stopsOf(portugal, ASPECT).map(({region}) => region)).toEqual(["Europe", "Africa", "Asia"])
    })

    it("are framed on their painted flags, never down in the tiles", () => {
        for (const {shot} of stopsOf(portugal, ASPECT)) {
            expect(shot.zoom).toBeGreaterThanOrEqual(1.5)
            expect(shot.zoom).toBeLessThanOrEqual(2.2)
        }
    })

    it("are where the flag took the most on a continent, not the middle of all it took there", () => {
        const africa = [...visits("Africa", 30, 0.2, 0, 1), ...visits("Africa", 10, 1.4, 0, 1)]

        expect(stopsOf(africa, ASPECT)[0].shot.direction.x).toBeCloseTo(Math.sin(0.22), 1)
    })

    it("are four at most", () => {
        const everywhere = ["Europe", "Africa", "Asia", "Oceania", "South America"].flatMap((region, i) => visits(region, 20, i, 0, 1))

        expect(stopsOf(everywhere, ASPECT)).toHaveLength(4)
    })
})

describe("a tour", () => {
    const stops = [
        {region: "Europe", shot: {direction: at(0), zoom: 2}, arrival: 0},
        {region: "Asia", shot: {direction: at(1.2), zoom: 2}, arrival: 0.5},
    ]
    const seconds = tourSecondsOf(2)
    const tour = tourOf(stops, ASPECT, seconds)
    const after = (time: number) => tour(time / seconds)

    it("opens on the whole globe over its first stop", () => {
        expect(after(0).zoom).toBeCloseTo(widestZoomOf(ASPECT))
        expect(after(0).direction.x).toBeCloseTo(0)
    })

    it("stops at each continent in turn, closing in while it is there", () => {
        expect(after(1.5).direction.x).toBeCloseTo(0, 1)
        expect(after(3).zoom).toBeGreaterThan(after(1.5).zoom)
        expect(after(5).direction.x).toBeCloseTo(Math.sin(1.2), 1)
    })

    it("flies high between continents far apart", () => {
        expect(after(4.05).zoom).toBeLessThan(2)
    })

    it("ends on the globe over all its stops", () => {
        expect(after(seconds).zoom).toBeCloseTo(widestZoomOf(ASPECT))
        expect(after(seconds).direction.x).toBeCloseTo(Math.sin(0.6), 1)
    })
})
