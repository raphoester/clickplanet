import {describe, expect, it} from "vitest"
import {buildSky, mulberry32} from "./stars.ts"

const SEED = 1234

function directions(positions: Float32Array): {x: number, y: number, z: number}[] {
    const out = []
    for (let i = 0; i < positions.length; i += 3) {
        const [x, y, z] = [positions[i], positions[i + 1], positions[i + 2]]
        const radius = Math.hypot(x, y, z)
        out.push({x: x / radius, y: y / radius, z: z / radius})
    }
    return out
}

describe("mulberry32", () => {
    it("gives the same sequence for the same seed", () => {
        const first = mulberry32(SEED)
        const second = mulberry32(SEED)

        const drawn = Array.from({length: 20}, () => first())

        expect(drawn).toEqual(Array.from({length: 20}, () => second()))
    })

    it("stays inside [0, 1)", () => {
        const random = mulberry32(SEED)

        for (let i = 0; i < 10_000; i++) {
            const value = random()
            expect(value).toBeGreaterThanOrEqual(0)
            expect(value).toBeLessThan(1)
        }
    })
})

describe("buildSky", () => {
    it("gives the same sky on every build, so it does not reshuffle on each mount", () => {
        const first = buildSky(500, SEED)
        const second = buildSky(500, SEED)

        expect(Array.from(first.positions)).toEqual(Array.from(second.positions))
        expect(Array.from(first.sizes)).toEqual(Array.from(second.sizes))
        expect(Array.from(first.tints)).toEqual(Array.from(second.tints))
    })

    it("puts every star on the sphere", () => {
        const {positions} = buildSky(2000, SEED)

        for (let i = 0; i < positions.length; i += 3) {
            const radius = Math.hypot(positions[i], positions[i + 1], positions[i + 2])
            expect(radius).toBeCloseTo(100, 2)
        }
    })

    it("spreads the stars evenly, rather than packing them round the poles", () => {
        const {positions} = buildSky(20_000, SEED)

        // Ten bands of equal area: equal height on a sphere is equal area, so
        // an even sky fills each of them about equally. A random latitude and a
        // random longitude would crowd the two end bands.
        const bands = new Array(10).fill(0)
        for (const {y} of directions(positions)) {
            bands[Math.min(9, Math.floor(((y + 1) / 2) * 10))]++
        }

        for (const count of bands) {
            expect(count).toBeGreaterThan(2000 * 0.9)
            expect(count).toBeLessThan(2000 * 1.1)
        }
    })

    it("spreads them evenly around the axis too", () => {
        const {positions} = buildSky(20_000, SEED)

        const quadrants = new Array(4).fill(0)
        for (const {x, z} of directions(positions)) {
            const angle = Math.atan2(z, x) + Math.PI
            quadrants[Math.min(3, Math.floor((angle / (2 * Math.PI)) * 4))]++
        }

        for (const count of quadrants) {
            expect(count).toBeGreaterThan(5000 * 0.9)
            expect(count).toBeLessThan(5000 * 1.1)
        }
    })

    it("keeps the stars small and faint enough to stay behind the game", () => {
        const {sizes, tints} = buildSky(5000, SEED)

        for (const size of sizes) {
            expect(size).toBeGreaterThanOrEqual(1.1)
            expect(size).toBeLessThanOrEqual(3)
        }
        for (const channel of tints) {
            expect(channel).toBeGreaterThan(0)
            expect(channel).toBeLessThan(0.9)
        }
    })

    it("varies both size and brightness, so the sky is not a regular grid", () => {
        const {sizes, tints} = buildSky(1000, SEED)

        expect(new Set(sizes).size).toBeGreaterThan(900)
        expect(new Set(Array.from({length: 1000}, (_, i) => tints[i * 3 + 1])).size).toBeGreaterThan(900)
    })
})
