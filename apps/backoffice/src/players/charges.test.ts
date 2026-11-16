import {describe, expect, it} from "vitest"
import {ChargesHeld} from "../gen/grpc/planet/v1/planet_pb.ts"
import {NO_GRANT} from "./actions.ts"
import {grantWords, heldWords, wholeOf} from "./charges.ts"

describe("grantWords", () => {
    it("lists every charge asked, in words", () => {
        expect(grantWords({refills: 1, bombs: 2, enclosures: 3, spreadClicks: 1, shields: 2, viruses: 1, mines: 2, rallies: 2}))
            .toBe("1 refill, 2 bombs, 3 enclosures, 1 spread click, 2 shields, 1 virus, 2 mines and 2 rallies")
        expect(grantWords({...NO_GRANT, refills: 3, viruses: 2})).toBe("3 refills and 2 viruses")
        expect(grantWords({...NO_GRANT, shields: 1})).toBe("1 shield")
        expect(grantWords(NO_GRANT)).toBe("nothing")
    })
})

describe("heldWords", () => {
    it("says what an account holds", () => {
        expect(heldWords(new ChargesHeld({bombs: 1, spreadClicksLeft: 8}))).toBe("1 bomb and 8 spread clicks")
        expect(heldWords(new ChargesHeld({refills: 2, bombs: 2, viruses: 3}))).toBe("2 refills, 2 bombs and 3 viruses")
        expect(heldWords(new ChargesHeld({viruses: 1}))).toBe("1 virus")
        expect(heldWords(new ChargesHeld({mines: 1}))).toBe("1 mine")
        expect(heldWords(new ChargesHeld({rallies: 1}))).toBe("1 rally")
        expect(heldWords(new ChargesHeld())).toBe("nothing")
    })
})

describe("wholeOf", () => {
    it("reads a count from 0 to 99", () => {
        expect(wholeOf("3")).toBe(3)
        expect(wholeOf("2.7")).toBe(2)
        expect(wholeOf("-4")).toBe(0)
        expect(wholeOf("")).toBe(0)
        expect(wholeOf("1000")).toBe(99)
    })
})
