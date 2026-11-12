import {describe, expect, it} from "vitest";
import {describeEnd} from "./endTime.ts";

const now = new Date(2026, 9, 20, 18, 0).getTime()

describe("describeEnd", () => {
    it("says the time alone under 20 hours", () => {
        const at = new Date(2026, 9, 21, 9, 40).getTime()

        expect(describeEnd(at, now)).toBe(new Intl.DateTimeFormat("en", {hour: "2-digit", minute: "2-digit"}).format(at))
    })

    it("says the day from 20 hours", () => {
        const at = new Date(2026, 9, 27, 18, 0).getTime()

        expect(describeEnd(at, now)).toBe("Oct 27, 06:00 PM")
    })

    it("says the year when it is not this one", () => {
        const at = new Date(2029, 9, 20, 18, 0).getTime()

        expect(describeEnd(at, now)).toContain("2029")
    })
})
