import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARD} from "../app/viewer/boardAsset.ts"
import {decodeRuns, tilesOf} from "./board.ts"
import {latticeSize} from "./lattice.ts"
import {decodeTerrain, encodeTerrain, KINDS, LAND, OPEN_SEA, seaKind} from "./terrain.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

function bytesOf(path: string): ArrayBuffer {
    const bytes = readFileSync(path)
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

const runOf = (kind: number, length: number) => new Array<number>(length).fill(kind)

const roundTrip = (kinds: Uint8Array) => {
    const bytes = encodeTerrain(kinds)
    return decodeTerrain(bytes.buffer as ArrayBuffer, kinds.length)
}

describe("encodeTerrain and decodeTerrain", () => {
    it("give back every point, whatever its run's length", () => {
        const kinds = Uint8Array.from([
            ...runOf(0, 1), ...runOf(3, 15), ...runOf(OPEN_SEA, 16), ...runOf(1, 17), ...runOf(seaKind(1), 143),
            ...runOf(seaKind(2), 144), ...runOf(7, 300_000), ...runOf(OPEN_SEA, 1),
        ])
        expect(roundTrip(kinds)).toEqual(kinds)
    })

    it("keep a run of 16 or more in one byte and a length after it", () => {
        expect(encodeTerrain(Uint8Array.from(runOf(2, 15))).length).toBe(12 + 1)
        expect(encodeTerrain(Uint8Array.from(runOf(2, 16))).length).toBe(12 + 2)
        expect(encodeTerrain(Uint8Array.from(runOf(2, 15 + 128 + 1))).length).toBe(12 + 3)
    })

    it("refuse a kind past the last", () => {
        expect(() => encodeTerrain(Uint8Array.from([KINDS]))).toThrow(/past the 12 kinds/)
        const bytes = encodeTerrain(Uint8Array.from([0]))
        bytes[12] = KINDS << 4
        expect(() => decodeTerrain(bytes.buffer as ArrayBuffer, 1)).toThrow(/past the 12 kinds/)
    })

    it("refuse a file of another lattice, or one whose runs do not fill it", () => {
        const bytes = encodeTerrain(Uint8Array.from(runOf(OPEN_SEA, 20)))
        expect(() => decodeTerrain(bytes.buffer as ArrayBuffer, 21)).toThrow(/of 20 points, the lattice has 21/)
        new DataView(bytes.buffer).setUint32(8, 21, true)
        expect(() => decodeTerrain(bytes.buffer as ArrayBuffer, 21)).toThrow(/hold 20 points/)
        new DataView(bytes.buffer).setUint32(8, 19, true)
        expect(() => decodeTerrain(bytes.buffer as ArrayBuffer, 19)).toThrow(/more than the lattice's 19/)
    })

    it("refuse a file cut in a run's length", () => {
        const bytes = encodeTerrain(Uint8Array.from(runOf(1, 200)))
        expect(() => decodeTerrain(bytes.buffer.slice(0, bytes.length - 1) as ArrayBuffer, 200)).toThrow(/truncated/)
    })

    it("refuse what is not a terrain file", () => {
        expect(() => decodeTerrain(new TextEncoder().encode("CPRN0000000000").buffer as ArrayBuffer, 1)).toThrow(/not a terrain file/)
    })
})

describe("the shipped terrain", () => {
    const terrain = decodeTerrain(bytesOf(`${STATIC}boards/${BOARD.name}/${BOARD.files.terrain}`), latticeSize(BOARD.lattice.cols))
    const runs = decodeRuns(bytesOf(`${STATIC}boards/${BOARD.name}/${BOARD.files.runs}`))

    it("has land exactly under the board's tiles, which the territory layer reads a tile's cell by", () => {
        const isTile = new Uint8Array(terrain.length)
        for (const id of tilesOf(BOARD.lattice, runs, BOARD.tiles).ids) isTile[id] = 1
        let astray = 0
        for (let id = 0; id < terrain.length; id++) if ((terrain[id] < LAND.length) !== (isTile[id] === 1)) astray++
        expect(astray).toBe(0)
    })
})
