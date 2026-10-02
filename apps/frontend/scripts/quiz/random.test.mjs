import {describe, expect, it} from "vitest"
import {first, pick, randomFrom, shuffled} from "./random.mjs"

const letters = ["a", "b", "c", "d", "e", "f", "g", "h"]

describe("the seeded draw", () => {
    // The bank is committed and content-addressed. A generator that drew with Math.random would
    // write a different file from the same data, and every regeneration would look like a change.
    it("is the same every run for the same seed", () => {
        expect(shuffled(letters, "capital:ee")).toEqual(shuffled(letters, "capital:ee"))
        expect(pick(letters, 3, "borders:np")).toEqual(pick(letters, 3, "borders:np"))
    })

    it("is a different draw for a different seed", () => {
        // Adding a country moves the questions about that country, and nothing else.
        expect(shuffled(letters, "capital:ee")).not.toEqual(shuffled(letters, "capital:lv"))
    })

    it("keeps every item, and only those", () => {
        expect([...shuffled(letters, "s")].sort()).toEqual([...letters].sort())
    })

    it("is not the order it was given", () => {
        expect(shuffled(letters, "s")).not.toEqual(letters)
    })
})

describe("first", () => {
    it("keeps the order it was given", () => {
        // The templates rank their candidates by how good a wrong answer is — same subregion, then
        // same continent — and a reshuffle here would throw that away.
        expect(first(letters, 3)).toEqual(["a", "b", "c"])
    })

    it("drops repeats and blanks", () => {
        // Two countries can share the name of a capital, and a country can have no capital at all.
        expect(first(["a", "a", undefined, "", "b"], 3)).toEqual(["a", "b"])
    })

    it("hands back what there is when there is not enough", () => {
        expect(first(["a"], 5)).toEqual(["a"])
    })

    it("hands back nothing for nothing", () => {
        expect(first([], 5)).toEqual([])
    })
})

describe("pick", () => {
    it("draws distinct items", () => {
        const drawn = pick(["a", "a", "b", "c"], 3, "s")
        expect(new Set(drawn).size).toBe(drawn.length)
    })

    it("takes at most what was asked for", () => {
        expect(pick(letters, 2, "s")).toHaveLength(2)
    })
})

describe("randomFrom", () => {
    it("stays inside [0, 1)", () => {
        const random = randomFrom("s")
        for (let i = 0; i < 1000; i++) {
            const drawn = random()
            expect(drawn).toBeGreaterThanOrEqual(0)
            expect(drawn).toBeLessThan(1)
        }
    })
})
