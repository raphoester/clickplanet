import {describe, expect, it} from "vitest"
import {describeEnd} from "./endTime.ts"
import {heldLine} from "./heldLine.ts"

describe("heldLine", () => {
    const now = 1_700_000_000_000

    it("names the flag held and when another may be played", () => {
        const changeableAt = now + 30 * 60_000

        expect(heldLine({countryId: "de", changeableAt}, now)).toBe(`You play for Germany until ${describeEnd(changeableAt, now)}`)
        expect(heldLine({countryId: "de"}, now)).toBe("You play for Germany")
        expect(heldLine({countryId: "de", changeableAt: now - 1}, now)).toBe("You play for Germany")
    })

    it("names no time while the rules lock the flag", () => {
        expect(heldLine({countryId: "de", changeableAt: now + 30 * 60_000, locked: true}, now)).toBe("You play for Germany")
    })
})
