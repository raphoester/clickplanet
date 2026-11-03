import {describe, expect, it} from "vitest"
import {linkedShot, withoutLinkedShot} from "./linkedShot.ts"

function page(path: string): URL {
    return new URL(path, "https://clickplanet.lol")
}

describe("linkedShot", () => {
    it("looks at the place, at the link's zoom", () => {
        const shot = linkedShot(page("/play?lat=40.42&lon=-100.00&zoom=3.0"))!

        expect(shot.zoom).toBe(3)
        expect(Math.asin(shot.direction.y) * 180 / Math.PI).toBeCloseTo(40.42, 9)
        expect(Math.atan2(shot.direction.z, -shot.direction.x) * 180 / Math.PI).toBeCloseTo(-100, 9)
    })

    it("puts the north pole on y and longitude 0 on -x, as the tile positions", () => {
        const directionAt = (lat: number, lon: number) => linkedShot(page(`/play?lat=${lat}&lon=${lon}&zoom=1`))!.direction

        expect(directionAt(90, 0).y).toBeCloseTo(1, 9)
        expect(directionAt(0, 0).x).toBeCloseTo(-1, 9)
        expect(directionAt(0, 90).z).toBeCloseTo(1, 9)
    })

    it("reads no shot from a link that lacks a value or holds one off the globe", () => {
        for (const path of [
            "/play", "/play?lat=40&lon=-100", "/play?lat=40&lon=-100&zoom=", "/play?lat=north&lon=-100&zoom=3",
            "/play?lat=91&lon=-100&zoom=3", "/play?lat=40&lon=181&zoom=3", "/play?lat=40&lon=-100&zoom=0",
        ]) {
            expect(linkedShot(page(path)), path).toBeUndefined()
        }
    })

    it("drops the place from the address and keeps the rest of it", () => {
        expect(withoutLinkedShot(page("/play?lat=40&lon=-100&zoom=3&gfx=halo#top")).href)
            .toBe("https://clickplanet.lol/play?gfx=halo#top")
    })
})
