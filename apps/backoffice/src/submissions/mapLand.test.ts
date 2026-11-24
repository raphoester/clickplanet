import {describe, expect, it} from "vitest"
import {EARTH_1, FIXTURE_1, placesOn} from "../boards/testBoards.ts"
import {patch} from "../graffiti/testPlaces.ts"
import {landOf, WORLD_BOX, worldLandOf} from "./mapLand.ts"

const share = (land: Uint8Array) => land.reduce((sum, pixel) => sum + pixel, 0) / land.length

describe("landOf", () => {
    it("draws the tiles as land with no hole between them, and the sea past them", () => {
        const {places} = patch(41.39, 2.17, 12)
        // 40 px a degree, 12° square, the patch's middle in the box's: the patch is 2.8° from its middle to its edges.
        const box = {west: 2.17 - 6, north: 41.39 + 6, scale: 40, width: 480, height: 480}
        const land = landOf(places, box)
        const at = (longitude: number, latitude: number) =>
            land[Math.floor((box.north - latitude) * box.scale) * box.width + Math.floor((longitude - box.west) * box.scale)]
        for (let dx = -1; dx <= 1; dx += 0.05) {
            for (let dy = -1; dy <= 1; dy += 0.05) expect(at(2.17 + dx, 41.39 + dy)).toBe(1)
        }
        expect(at(2.17 + 5.5, 41.39)).toBe(0)
        expect(at(2.17, 41.39 - 3.5)).toBe(0)
    })

    it("draws round the world, past the antimeridian, whichever turn the box is on", () => {
        const {places} = patch(0, 179.9, 6)
        for (const west of [179, -181, 539]) {
            const land = landOf(places, {west, north: 1, scale: 20, width: 40, height: 40})
            expect(land[20 * 40 + 18]).toBe(1)
            expect(land[20 * 40 + 22]).toBe(1)
        }
    })

    it("draws a tile smaller than a pixel on the pixel it is on", () => {
        const {places} = patch(41.39, 2.17, 1)
        const land = landOf(places, WORLD_BOX)
        expect(land[Math.floor((90 - 41.39) * 4) * 1440 + Math.floor((2.17 + 180) * 4)]).toBe(1)
        expect(land.reduce((sum, pixel) => sum + pixel, 0)).toBeLessThan(20)
    })

    it("draws a coarse board's world as whole as earth-1's: each tile reaches by its board's step", () => {
        const earth = share(worldLandOf(placesOn(EARTH_1)))
        const coarse = share(worldLandOf(placesOn(FIXTURE_1)))
        expect(earth).toBeGreaterThan(0.3)
        expect(earth).toBeLessThan(0.4)
        expect(Math.abs(coarse - earth)).toBeLessThan(0.06)
    })
})
