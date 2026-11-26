import {describe, expect, it} from "vitest"
import {count, daysOf, momentOf, percent, rate, secondsOf, viewsOf} from "./format.ts"

describe("format", () => {
    it("writes a span of UTC days short", () => {
        expect(daysOf("2026-11-20", "2026-11-20")).toBe("20 Nov")
        expect(daysOf("2026-11-20", "2026-11-23")).toBe("20–23 Nov")
        expect(daysOf("2026-11-30", "2026-12-02")).toBe("30 Nov – 2 Dec")
    })

    it("writes a post's moment in UTC", () => {
        expect(momentOf(new Date("2026-11-22T13:48:00Z"))).toBe("22 Nov 13:48")
    })

    it("writes views in full under 10,000 and compact past it", () => {
        expect(viewsOf(6940)).toBe("6,940")
        expect(viewsOf(12_617)).toBe("12.6K")
        expect(viewsOf(undefined)).toBe("—")
    })

    it("writes no value as a dash", () => {
        expect([count(undefined), percent(undefined), secondsOf(undefined), rate(undefined)]).toEqual(["—", "—", "—", "—"])
        expect([count(1262.4), percent(0.1609), secondsOf(7.39), rate(3.816)]).toEqual(["1,262", "16%", "7.4 s", "3.8"])
    })
})
