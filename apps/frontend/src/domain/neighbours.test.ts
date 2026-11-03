import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {rebuildTilePositions} from "./landmask.ts"
import {Neighbours} from "./neighbours.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))

function shippedPositions(): Float32Array {
    const [name] = readdirSync(STATIC).filter((entry) => /^landmask-[0-9a-f]{8}\.bin$/.test(entry))
    const bytes = readFileSync(STATIC + name)
    return rebuildTilePositions(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength))
}

describe("Neighbours", () => {
    it("finds the tiles within reach and not the tile itself", () => {
        const positions = new Float32Array([
            1, 0, 0,
            Math.cos(0.004), Math.sin(0.004), 0,
            Math.cos(0.008), Math.sin(0.008), 0,
            0, 1, 0,
        ])
        const neighbours = new Neighbours(positions)

        expect(neighbours.of(1)).toEqual([2])
        expect(neighbours.of(2).sort()).toEqual([1, 3])
        expect(neighbours.of(4)).toEqual([])
    })

    it("gives an inland tile of the shipped map its six around, and no tile more", () => {
        const positions = shippedPositions()
        const neighbours = new Neighbours(positions)
        const counts: number[] = []
        for (let tile = 1; tile * 3 <= positions.length; tile += 7) counts.push(neighbours.of(tile).length)

        expect(Math.max(...counts)).toBe(6)
        expect(counts.filter((count) => count === 6).length / counts.length).toBeGreaterThan(0.9)
    })
})
