import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARDS} from "../app/viewer/boardsAsset.ts"
import {bordersHashOf, decodeProvinces, encodeProvinces, type Provinces} from "./provinces.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

function bytesOf(path: string): ArrayBuffer {
    const bytes = readFileSync(path)
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

const sample = (): Provinces => ({
    borders: 0x0ae1a037,
    province: Uint16Array.of(1, 1, 2),
    landmass: Uint16Array.of(0, 3, 3),
    frames: Float32Array.of(0, 0, 0, 0, 0, 1, 0, 0, 0.1, 0.2, 0, 1, 0, 0.3, 0.4),
    tiles: Uint32Array.of(0, 2, 1),
})

const copy = (bytes: Uint8Array): ArrayBuffer => bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer

describe("a provinces file", () => {
    it("reads back what it was written from", () => {
        const decoded = decodeProvinces(copy(encodeProvinces(sample())), 3)

        expect(decoded.borders).toBe(0x0ae1a037)
        expect([...decoded.province]).toEqual([1, 1, 2])
        expect([...decoded.landmass]).toEqual([0, 3, 3])
        expect([...decoded.tiles]).toEqual([0, 2, 1])
        expect([...decoded.frames]).toEqual([...sample().frames])
    })

    it("is refused for another board's tile count, or a tile in no province", () => {
        const bytes = copy(encodeProvinces(sample()))
        expect(() => decodeProvinces(bytes, 4)).toThrow("the board has 4")

        const none = new Uint8Array(bytes.slice(0))
        none[none.length - 2] = 0
        expect(() => decodeProvinces(none.buffer, 3)).toThrow("tile 3 is in province 0")
        expect(() => encodeProvinces({...sample(), province: Uint16Array.of(1, 1, 3)})).toThrow("province 3")
    })

    it("names its borders by the 8 hex of their file", () => {
        expect(bordersHashOf("borders-0ae1a037.bin")).toBe(0x0ae1a037)
        expect(() => bordersHashOf("terrain-0ae1a037.bin")).toThrow()
    })
})

describe("the shipped provinces", () => {
    for (const board of Object.values(BOARDS)) {
        if (board.files.provinces === undefined) continue
        const directory = `${STATIC}boards/${board.name}/`

        it(`put every tile of ${board.name} in a province of its landmass, cut on its borders`, () => {
            const provinces = decodeProvinces(bytesOf(directory + board.files.provinces), board.tiles)
            expect(provinces.borders).toBe(bordersHashOf(board.files.borders))

            const bordersBytes = readFileSync(directory + board.files.borders)
            const header = bordersBytes.readUInt32LE(0)
            const landmass = new Uint16Array(bordersBytes.buffer.slice(bordersBytes.byteOffset + 4 + header, bordersBytes.byteOffset + 4 + header + board.tiles * 2))
            const counted = new Uint32Array(provinces.landmass.length)
            for (let t = 0; t < board.tiles; t++) {
                expect(provinces.landmass[provinces.province[t]]).toBe(landmass[t])
                counted[provinces.province[t]]++
            }
            expect([...counted]).toEqual([...provinces.tiles])
        })

        it(`name every province of ${board.name} but a country's whole main landmass, keyed by its provinces file`, () => {
            const names = JSON.parse(readFileSync(directory + board.page.provinceNames!, "utf8"))
            expect(names.provinces).toBe(board.files.provinces)
            const provinces = decodeProvinces(bytesOf(directory + board.files.provinces), board.tiles)
            const landmasses = JSON.parse(readFileSync(directory + board.page.landmassNames, "utf8")).names
            const onLandmass = new Map<number, number>()
            for (const landmass of provinces.landmass.subarray(1)) onLandmass.set(landmass, (onLandmass.get(landmass) ?? 0) + 1)
            const unnamed = Array.from({length: provinces.landmass.length - 1}, (_, k) => k + 1).filter((k) => !(String(k) in names.names))
            expect(unnamed.filter((k) => onLandmass.get(provinces.landmass[k]) !== 1 || String(provinces.landmass[k]) in landmasses)).toEqual([])
            expect(Object.keys(names.names).map(Number).filter((k) => !(k >= 1 && k < provinces.landmass.length))).toEqual([])
        })
    }

    it("are on earth-2 and siege-1, and not on earth-1", () => {
        expect(Object.values(BOARDS).filter((board) => board.files.provinces).map((board) => board.name).sort()).toEqual(["earth-2", "siege-1"])
    })
})
