import {describe, expect, it} from "vitest"
import {ChargesHeld} from "../gen/grpc/planet/v1/planet_pb.ts"
import {NO_GRANT} from "./actions.ts"
import {grantWords, heldWords, wholeOf} from "./charges.ts"

describe("grantWords", () => {
    it("lists every charge asked, in words", () => {
        expect(grantWords({refill: true, bomb: true, enclosures: 3, spreadClicks: 1, shields: 2}))
            .toBe("a refill, a bomb, 3 enclosures, 1 spread click and 2 shields")
        expect(grantWords({...NO_GRANT, shields: 1})).toBe("1 shield")
        expect(grantWords(NO_GRANT)).toBe("nothing")
    })
})

describe("heldWords", () => {
    it("says what an account holds", () => {
        expect(heldWords(new ChargesHeld({bomb: true, spreadClicksLeft: 8}))).toBe("a bomb and 8 spread clicks")
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
