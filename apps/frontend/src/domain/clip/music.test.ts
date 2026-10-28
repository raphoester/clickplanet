import {describe, expect, it} from "vitest"
import {startOf} from "./music.ts"

describe("where a clip starts its anthem", () => {
    const anthem = {seconds: 60, at: [30, 45, 11]}

    it("is the best highlight", () => {
        expect(startOf(anthem, 12)).toBe(30)
    })

    it("skips a highlight too late for the anthem to last the whole clip", () => {
        expect(startOf({seconds: 60, at: [45, 30, 11]}, 20)).toBe(30)
        expect(startOf(anthem, 40)).toBe(11)
    })

    it("is early enough for the anthem to last the whole clip when no highlight does", () => {
        expect(startOf({seconds: 30, at: [14]}, 20)).toBe(10)
        expect(startOf(anthem, 70)).toBe(0)
    })

    it("is the start of an anthem nobody measured", () => {
        expect(startOf(undefined, 12)).toBe(0)
        expect(startOf({seconds: 60, at: []}, 12)).toBe(0)
    })
})
