import {describe, expect, it} from "vitest"
import {heldLine, lockOf, minutesLeftOf} from "./allegiance.ts"
import {describeEnd} from "./endTime.ts"

describe("lockOf", () => {
    const now = 1_700_000_000_000

    it("holds a flag until the time the server said", () => {
        expect(lockOf({countryId: "de", changeableAt: now + 1}, now)).toEqual({kind: "wait", until: now + 1})
    })

    it("holds nothing once that time came, for a flag never changed, or with no flag", () => {
        expect(lockOf({countryId: "de", changeableAt: now}, now)).toBeUndefined()
        expect(lockOf({countryId: "fr"}, now)).toBeUndefined()
        expect(lockOf(undefined, now)).toBeUndefined()
    })

    it("holds a flag the rules lock, with no time, whatever the wait says", () => {
        expect(lockOf({countryId: "de", locked: true}, now)).toEqual({kind: "rules"})
        expect(lockOf({countryId: "de", changeableAt: now + 1, locked: true}, now)).toEqual({kind: "rules"})
        expect(lockOf({countryId: "de", changeableAt: now - 1, locked: true}, now)).toEqual({kind: "rules"})
    })
})

describe("minutesLeftOf", () => {
    const now = 1_700_000_000_000

    it("counts a minute begun as a whole one, and never less than one", () => {
        expect(minutesLeftOf(now + 51 * 60_000 + 1, now)).toBe(52)
        expect(minutesLeftOf(now + 60 * 60_000, now)).toBe(60)
        expect(minutesLeftOf(now + 5_000, now)).toBe(1)
        expect(minutesLeftOf(now, now)).toBe(1)
    })
})

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
