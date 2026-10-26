import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {decodeCoordinates} from "../app/viewer/coordinatesBinary.ts"
import {LANDMASK_URL} from "../app/viewer/landmaskAsset.ts"
import {latticePositions, latticeSize, positionsHash} from "./lattice.ts"
import {
    decodeLandmask,
    encodeLandmask,
    LANDMASK_HEADER_BYTES,
    landmaskOf,
    PositionsMismatchError,
    rebuildTilePositions,
} from "./landmask.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

function staticFile(pattern: RegExp): ArrayBuffer {
    const names = readdirSync(STATIC).filter((name) => pattern.test(name))
    expect(names).toHaveLength(1)
    const bytes = readFileSync(STATIC + names[0])
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

function firstPoints(detail: number, count: number): Float32Array {
    return latticePositions(detail).slice(0, count * 3)
}

describe("the shipped landmask", () => {
    it("rebuilds every tile of the shipped coordinates blob bit for bit, in tile order", () => {
        const blob = decodeCoordinates(staticFile(/^coordinates-[0-9a-f]{8}\.bin$/)).positions
        const rebuilt = rebuildTilePositions(staticFile(/^landmask-[0-9a-f]{8}\.bin$/))

        expect(rebuilt.length).toBe(blob.length)
        const want = new Uint32Array(blob.buffer, blob.byteOffset, blob.length)
        const got = new Uint32Array(rebuilt.buffer)
        let differing = 0
        for (let k = 0; k < want.length; k++) if (got[k] !== want[k]) differing++
        expect(differing).toBe(0)
    })

    it("is the file the app fetches", () => {
        const [name] = readdirSync(STATIC).filter((entry) => /^landmask-[0-9a-f]{8}\.bin$/.test(entry))
        expect(LANDMASK_URL).toBe(`/static/${name}`)
    })
})

describe("landmaskOf", () => {
    it("marks the points the tiles sit on", () => {
        const all = latticePositions(4)
        const tiles = Float32Array.from([...all.subarray(3, 6), ...all.subarray(30, 33)])

        const {isTile, tiles: count, detail} = landmaskOf(4, tiles)

        expect([detail, count]).toEqual([4, 2])
        expect(isTile[0]).toBe(0b10)
        expect(isTile[1]).toBe(0b100)
        expect(isTile.slice(2).every((byte) => byte === 0)).toBe(true)
    })

    it("refuses a tile that is not on the lattice", () => {
        const tiles = firstPoints(4, 3)
        tiles[4] += 0.001
        expect(() => landmaskOf(4, tiles)).toThrow(/tile 2/)
    })

    it("refuses tiles out of lattice order", () => {
        const all = latticePositions(4)
        const tiles = Float32Array.from([...all.subarray(30, 33), ...all.subarray(3, 6)])
        expect(() => landmaskOf(4, tiles)).toThrow(/tile 2/)
    })
})

describe("encodeLandmask and decodeLandmask", () => {
    it("round-trip", () => {
        const landmask = landmaskOf(5, firstPoints(5, 40))
        expect(decodeLandmask(encodeLandmask(landmask))).toEqual(landmask)
    })

    it("refuses a file that is not a landmask", () => {
        const buffer = encodeLandmask(landmaskOf(2, firstPoints(2, 3)))
        new DataView(buffer).setUint8(0, 0)
        expect(() => decodeLandmask(buffer)).toThrow(/not a landmask file/)
    })

    it("refuses another format version", () => {
        const buffer = encodeLandmask(landmaskOf(2, firstPoints(2, 3)))
        new DataView(buffer).setUint32(4, 2, true)
        expect(() => decodeLandmask(buffer)).toThrow(/version 2/)
    })

    it("refuses a point count that is not the detail's", () => {
        const buffer = encodeLandmask(landmaskOf(2, firstPoints(2, 3)))
        new DataView(buffer).setUint32(12, latticeSize(2) + 1, true)
        expect(() => decodeLandmask(buffer)).toThrow(/points/)
    })

    it("refuses a truncated file", () => {
        const buffer = encodeLandmask(landmaskOf(2, firstPoints(2, 3)))
        expect(() => decodeLandmask(buffer.slice(0, buffer.byteLength - 1))).toThrow(/size mismatch/)
        expect(() => decodeLandmask(buffer.slice(0, LANDMASK_HEADER_BYTES - 1))).toThrow(/truncated/)
    })
})

describe("rebuildTilePositions", () => {
    it("hands back the positions the landmask was made from", () => {
        const tiles = firstPoints(10, 50)
        const rebuilt = rebuildTilePositions(encodeLandmask(landmaskOf(10, tiles)))
        expect(positionsHash(rebuilt)).toBe(positionsHash(tiles))
    })

    it("refuses positions that do not hash to what the file says", () => {
        const landmask = landmaskOf(3, firstPoints(3, 10))
        expect(() => rebuildTilePositions(encodeLandmask({...landmask, hash: landmask.hash ^ 1})))
            .toThrow(PositionsMismatchError)
    })
})
