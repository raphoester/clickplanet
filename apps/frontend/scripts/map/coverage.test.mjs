import {describe, expect, it} from "vitest"

import {coverage, spacingOf, tilesOn} from "./coverage.mjs"
import {lattice, neighbours} from "./lattice.mjs"

const DETAIL = 10
const width = 360
const height = 180

const vertices = lattice(DETAIL)
const edges = neighbours(DETAIL)
const spacing = spacingOf(vertices.positions, edges)

const lonLatOf = (v) => [
    Math.atan2(vertices.positions[v * 3 + 2], -vertices.positions[v * 3]) * 180 / Math.PI,
    Math.asin(vertices.positions[v * 3 + 1]) * 180 / Math.PI,
]

const pixelOf = (lon, lat) =>
    Math.min(Math.floor((90 - lat) / 180 * height), height - 1) * width
    + Math.min(Math.floor((lon + 180) / 360 * width), width - 1)

const coverWith = (isTile) => coverage(vertices, isTile, spacing, width, height)

describe("coverage", () => {
    it("is land where the nearest vertex is a tile and sea where it is not", () => {
        const isTile = new Uint8Array(vertices.count).map((_, v) => v % 3 === 0 ? 1 : 0)
        const cover = coverWith(isTile)

        for (let v = 0; v < vertices.count; v++) {
            expect(cover[pixelOf(...lonLatOf(v))]).toBe(isTile[v])
        }
    })

    it("makes a lake of a sea vertex with tiles all round it", () => {
        const lake = vertices.count >> 1
        const isTile = new Uint8Array(vertices.count).fill(1)
        isTile[lake] = 0
        const cover = coverWith(isTile)

        const centre = vertices.positions.subarray(lake * 3, lake * 3 + 3)
        const water = []
        for (let y = 0; y < height; y++) {
            const lat = (90 - (y + 0.5) / height * 180) * Math.PI / 180
            for (let x = 0; x < width; x++) {
                const lon = ((x + 0.5) / width * 360 - 180) * Math.PI / 180
                const along = -Math.cos(lat) * Math.cos(lon) * centre[0] + Math.sin(lat) * centre[1]
                    + Math.cos(lat) * Math.sin(lon) * centre[2]
                if (Math.acos(Math.min(1, along)) < spacing / 5) water.push(cover[y * width + x])
            }
        }
        expect(water.length).toBeGreaterThan(1)
        expect(water.every((value) => value === 0)).toBe(true)
        for (let e = edges.at[lake]; e < edges.at[lake + 1]; e++) {
            expect(cover[pixelOf(...lonLatOf(edges.to[e]))]).toBe(1)
        }
    })

    it("is half way at the cell edge, and fades across it", () => {
        const tile = vertices.count >> 1
        const isTile = new Uint8Array(vertices.count)
        isTile[tile] = 1
        const cover = coverWith(isTile)

        const sea = edges.to[edges.at[tile]]
        const [lon, lat] = lonLatOf(tile)
        const [seaLon, seaLat] = lonLatOf(sea)
        const edge = cover[pixelOf((lon + seaLon) / 2, (lat + seaLat) / 2)]

        expect(edge).toBeGreaterThan(0.2)
        expect(edge).toBeLessThan(0.8)
        expect([...cover].some((value) => value > 0 && value < 1)).toBe(true)
    })

    it("wraps a tile on the antimeridian into both edges", () => {
        let tile = -1
        for (let v = 0; v < vertices.count; v++) {
            const [lon, lat] = lonLatOf(v)
            if (Math.abs(lat) < 30 && (tile < 0 || Math.abs(lon) > Math.abs(lonLatOf(tile)[0]))) tile = v
        }
        const isTile = new Uint8Array(vertices.count)
        isTile[tile] = 1
        const cover = coverWith(isTile)
        const row = pixelOf(0, lonLatOf(tile)[1]) - width / 2

        expect(cover[row]).toBeGreaterThan(0)
        expect(cover[row + width - 1]).toBeGreaterThan(0)
    })
})

describe("tilesOn", () => {
    it("marks the vertex under each tile", () => {
        const picked = [0, 7, vertices.count - 1]
        const tiles = {
            count: picked.length,
            positions: Float32Array.from(picked.flatMap((v) => [...vertices.positions.subarray(v * 3, v * 3 + 3)])),
        }

        const isTile = tilesOn(vertices, tiles)

        expect([...isTile.keys()].filter((v) => isTile[v])).toEqual(picked)
    })

    it("refuses a tile that is not on the lattice", () => {
        const tiles = {count: 1, positions: Float32Array.from([0.1, 0.2, 0.3])}

        expect(() => tilesOn(vertices, tiles)).toThrow(/not a lattice vertex/)
    })
})

describe("spacingOf", () => {
    it("is the mean arc between two touching lattice vertices", () => {
        const expected = Math.sqrt(4 * Math.PI / (20 * (DETAIL + 1) ** 2) * 4 / Math.sqrt(3))
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
