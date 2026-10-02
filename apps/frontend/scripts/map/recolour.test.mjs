import {describe, expect, it} from "vitest"

import {recolour} from "./recolour.mjs"

const LAND = [60, 120, 50]
const SEA = [20, 40, 130]

function world({width = 256, height = 128, paint = (x) => x < width / 2} = {}) {
    const photo = new Uint8Array(width * height * 3)
    const cover = new Float32Array(width * height)

    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const at = y * width + x
            const colour = paint(x, y) ? LAND : SEA
            cover[at] = paint(x, y) ? 1 : 0
            for (let c = 0; c < 3; c++) photo[at * 3 + c] = colour[c]
        }
    }

    return {photo, cover, size: {width, height, channels: 3}}
}

const pixelAt = (pixels, {width}, x, y) => [0, 1, 2].map((c) => pixels[(y * width + x) * 3 + c])
const near = (a, b, slack = 8) => a.every((v, i) => Math.abs(v - b[i]) <= slack)

describe("recolour", () => {
    it("leaves a photo that already agrees with the tile field alone", () => {
        const {photo, cover, size} = world()
        const {pixels, moved} = recolour(photo, cover, size)

        expect(moved).toBe(0)
        expect(pixelAt(pixels, size, 40, 64)).toEqual(LAND)
        expect(pixelAt(pixels, size, 200, 64)).toEqual(SEA)
    })

    it("turns water the tile field covers into land", () => {
        const {photo, cover, size} = world()
        for (let y = 60; y < 68; y++) for (let x = 200; x < 208; x++) cover[y * size.width + x] = 1

        const {pixels} = recolour(photo, cover, size)

        expect(near(pixelAt(pixels, size, 204, 64), LAND)).toBe(true)
        expect(pixelAt(pixels, size, 220, 64)).toEqual(SEA)
    })

    it("turns land the tile field does not cover into water", () => {
        const {photo, cover, size} = world()
        for (let y = 60; y < 68; y++) for (let x = 40; x < 48; x++) cover[y * size.width + x] = 0

        const {pixels} = recolour(photo, cover, size)

        expect(near(pixelAt(pixels, size, 44, 64), SEA)).toBe(true)
        expect(pixelAt(pixels, size, 20, 64)).toEqual(LAND)
    })

    it("counts what it moved", () => {
        const {photo, cover, size} = world()
        for (let y = 60; y < 68; y++) for (let x = 200; x < 208; x++) cover[y * size.width + x] = 1

        const {moved} = recolour(photo, cover, size)

        expect(moved).toBeGreaterThan(0)
        expect(moved).toBeLessThan(size.width * size.height / 4)
    })

    it("changes nothing where land and water look the same", () => {
        const {photo, cover, size} = world({paint: () => true})
        for (let y = 0; y < size.height; y++) for (let x = 128; x < 256; x++) cover[y * size.width + x] = 0

        const {pixels, moved} = recolour(photo, cover, size)

        expect(moved).toBe(0)
        expect(pixelAt(pixels, size, 200, 64)).toEqual(LAND)
    })

    it("keeps what the photo says beyond the two colours", () => {
        const {photo, cover, size} = world()
        const at = (64 * size.width + 40) * 3
        photo[at] = 200

        const {pixels} = recolour(photo, cover, size)

        expect(pixelAt(pixels, size, 40, 64)[0]).toBeGreaterThan(150)
    })
})
