import {describe, expect, it} from "vitest"
import {TileChange} from "./changes.ts"
import {frontOf, inSightOf, sameFront, spanOf} from "./front.ts"
import {scoreOf} from "./score.ts"

function took(from: string | undefined, count: number, first = 1): TileChange[] {
    return Array.from({length: count}, (_, i) => ({tile: first + i, from, to: "ps", at: i}))
}

describe("the score of a story", () => {
    it("counts tiles taken from another flag, not off empty ground", () => {
        expect(scoreOf(took("fr", 10), () => "fr", 1)).toBe(10)
        expect(scoreOf(took(undefined, 10), () => "fr", 1)).toBe(0)
    })

    it("rates a short window over a long one", () => {
        expect(scoreOf(took("fr", 100), () => "fr", 1)).toBeGreaterThan(scoreOf(took("fr", 100), () => "fr", 4))
    })

    it("rates a war over several countries over one", () => {
        const ground = (tile: number) => tile <= 50 ? "fr" : "de"

        expect(scoreOf(took("fr", 100), ground, 1)).toBe(200)
    })
})

describe("the front", () => {
    const point = (tile: number) => tile <= 3
        ? {x: 0, y: Math.sin(tile * 0.01), z: Math.cos(tile * 0.01)}
        : {x: 0, y: -1, z: 0}

    it("is where most tiles changed hands, and what is around it", () => {
        const front = frontOf(took("fr", 4), point)

        expect(front?.changes.map(({tile}) => tile)).toEqual([1, 2, 3])
    })

    it("is nothing when nothing changed hands", () => {
        expect(frontOf([], point)).toBeUndefined()
    })

    it("runs from its first change to its last", () => {
        const front = {heart: point(1), changes: took("fr", 3).map((change, i) => ({...change, at: 1000 * (i + 1)}))}

        expect(spanOf(front, 10)).toEqual({since: 990, until: 3010})
    })

    it("sees what is near it, on the country it is told on, and nothing across that country far away", () => {
        const front = {heart: point(1), changes: []}
        const ground = (tile: number) => tile === 2 ? "ca" : "us"

        expect(inSightOf(front, point, ground, "us")(1)).toBe(true)
        expect(inSightOf(front, point, ground, "us")(2)).toBe(false)
        expect(inSightOf(front, point, ground, "us")(9)).toBe(false)
        expect(inSightOf(front, point, ground, undefined)(2)).toBe(true)
    })

    it("is the same as one close by, and not as one across the world", () => {
        const here = {heart: point(1), changes: []}

        expect(sameFront(here, {heart: point(2), changes: []})).toBe(true)
        expect(sameFront(here, {heart: point(9), changes: []})).toBe(false)
    })
})
