import {describe, expect, it} from "vitest"

import {coverage, spacingOf} from "./coverage.mjs"
import {lattice, neighbours} from "./lattice.mjs"

// uv back from a longitude and a latitude, the way `lonLatOf` reads it.
const uvOf = (lon, lat) => [lon / 360 + 0.5, lat / 180 + 0.5]

const tilesAt = (...points) => ({
    count: points.length,
    uvs: Float32Array.from(points.flatMap(([lon, lat]) => uvOf(lon, lat))),
})

// Six degrees of arc, so one tile's cell is several pixels across on a 360x180 image and the tests
// are about the shape rather than about rounding.
const SPACING = 6 * Math.PI / 180

describe("coverage", () => {
    const width = 360
    const height = 180

    it("covers the tile's own pixel and nothing far from it", () => {
        const cover = coverage(tilesAt([0, 0]), SPACING, width, height)
        const at = (lon, lat) => cover[Math.round((90 - lat) / 180 * height) * width + Math.round((lon + 180) / 360 * width)]

        expect(at(0, 0)).toBe(1)
        expect(at(20, 0)).toBe(0)
        expect(at(0, 20)).toBe(0)
    })

    it("fades out rather than ending on an edge", () => {
        const cover = coverage(tilesAt([0, 0]), SPACING, width, height)
        const values = [...cover].filter((v) => v > 0)

        expect(values.length).toBeGreaterThan(1)
        expect(values.some((v) => v > 0 && v < 1)).toBe(true)
    })

    // A tile at 180 sits on the image's own cut, so its cell has to come out of both edges or the
    // dateline gets a seam of open water down it at every zoom.
    it("wraps a tile on the antimeridian into both edges", () => {
        const cover = coverage(tilesAt([179.9, 0]), SPACING, width, height)
        const row = Math.round(90 / 180 * height) * width

        expect(cover[row + width - 1]).toBeGreaterThan(0)
        expect(cover[row]).toBeGreaterThan(0)
    })

    // A cell is a fixed arc on the ground, and a column of an equirectangular image covers less
    // ground the further it is from the equator. A tile near a pole therefore has to paint a wider
    // ellipse, or the poles come out as bare water.
    it("paints a wider ellipse near a pole than at the equator", () => {
        const spread = (lat) => {
            const cover = coverage(tilesAt([0, lat]), SPACING, width, height)
            let columns = 0
            const row = Math.round((90 - lat) / 180 * height) * width
            for (let x = 0; x < width; x++) if (cover[row + x] > 0) columns++
            return columns
        }

        expect(spread(80)).toBeGreaterThan(spread(0) * 3)
    })

    it("covers more of the image the more tiles there are", () => {
        const one = coverage(tilesAt([0, 0]), SPACING, width, height).reduce((a, b) => a + b, 0)
        const two = coverage(tilesAt([0, 0], [40, 40]), SPACING, width, height).reduce((a, b) => a + b, 0)

        expect(two).toBeGreaterThan(one * 1.5)
    })
})

describe("spacingOf", () => {
    it("is the mean arc between two touching lattice vertices", () => {
        const detail = 10
        const {positions} = lattice(detail)
        const spacing = spacingOf(positions, neighbours(detail))

        // The sphere is cut into 20*(detail+1)^2 triangles, and an equilateral triangle of side s
        // covers s^2 * sqrt(3)/4, which puts s near this.
        const expected = Math.sqrt(4 * Math.PI / (20 * (detail + 1) ** 2) * 4 / Math.sqrt(3))
        expect(spacing).toBeGreaterThan(expected * 0.85)
        expect(spacing).toBeLessThan(expected * 1.15)
    })

    it("halves when the detail doubles", () => {
        const coarse = spacingOf(lattice(8).positions, neighbours(8))
        const fine = spacingOf(lattice(16).positions, neighbours(16))

        expect(fine).toBeGreaterThan(coarse * 0.45)
        expect(fine).toBeLessThan(coarse * 0.55)
    })
})
