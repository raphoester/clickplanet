import {describe, expect, it} from "vitest"
import {lockOf} from "./allegiance.ts"

describe("lockOf", () => {
    const now = 1_700_000_000_000

    it("holds a flag until the time the server said", () => {
        expect(lockOf({countryId: "de", changeableAt: now + 1}, now)).toEqual({countryId: "de", changeableAt: now + 1})
    })

    it("holds nothing once that time came, for a flag never changed, or with no flag", () => {
        expect(lockOf({countryId: "de", changeableAt: now}, now)).toBeUndefined()
        expect(lockOf({countryId: "fr"}, now)).toBeUndefined()
        expect(lockOf(undefined, now)).toBeUndefined()
    })
})
