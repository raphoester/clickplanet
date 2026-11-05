import {describe, expect, it} from "vitest"
import {spanOf, timeOf} from "./time.ts"

describe("a span of time", () => {
    it("says a day the two times share once", () => {
        const span = spanOf(new Date(2098, 5, 1, 22), new Date(2098, 5, 1, 23))
        expect(span.match(/2098/g)).toHaveLength(1)
    })

    it("says both days when they differ", () => {
        expect(spanOf(new Date(2098, 5, 1, 22), new Date(2098, 5, 2, 1)).match(/2098/g)).toHaveLength(2)
    })

    it("says the one time it has", () => {
        const at = new Date(2098, 5, 1, 22)
        expect(spanOf(at, undefined)).toBe(timeOf(at))
    })
})
