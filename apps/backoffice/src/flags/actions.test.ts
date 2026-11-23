import {describe, expect, it} from "vitest"
import {AT} from "../fakeAdmin.ts"
import {daysOf, leftOf} from "./actions.ts"

const HOUR = 3_600_000

describe("leftOf", () => {
    it("counts days to the nearest, then hours in the last two days", () => {
        expect(leftOf(new Date(AT.getTime() + 29.99 * 24 * HOUR), AT)).toBe("30 days left")
        expect(leftOf(new Date(AT.getTime() + 12.4 * 24 * HOUR), AT)).toBe("12 days left")
        expect(leftOf(new Date(AT.getTime() + 47.5 * HOUR), AT)).toBe("47 h left")
        expect(leftOf(new Date(AT.getTime() + 0.5 * HOUR), AT)).toBe("under an hour left")
    })

    it("says a flag past its end ends now, as the server closes it within the minute", () => {
        expect(leftOf(AT, AT)).toBe("ending now")
    })
})

describe("daysOf", () => {
    it("takes a whole number from 1 to 3660", () => {
        expect(daysOf(" 30 ")).toBe(30)
        expect(daysOf("3660")).toBe(3660)
        expect(daysOf("0")).toBeUndefined()
        expect(daysOf("3661")).toBeUndefined()
        expect(daysOf("1.5")).toBeUndefined()
        expect(daysOf("")).toBeUndefined()
    })
})
