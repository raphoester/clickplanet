import {describe, expect, it} from "vitest"
import {ALL_OFF, atRest, NO_CHARGES, switched, switchesHeld} from "./bonus.ts"

describe("switchesHeld", () => {
    const spread = {...ALL_OFF, spread: true}
    const enclose = {...ALL_OFF, enclose: true}
    const shield = {...ALL_OFF, shield: true}

    it("keeps a switch on while its pool holds something", () => {
        const held = {...NO_CHARGES, enclosures: 1, spreadClicksLeft: 4, shields: 2}

        expect(switchesHeld(spread, held, false)).toBe(spread)
        expect(switchesHeld(enclose, held, false)).toBe(enclose)
        expect(switchesHeld(shield, held, false)).toBe(shield)
    })

    it("turns a switch off once its pool is empty", () => {
        expect(switchesHeld(spread, {...NO_CHARGES, enclosures: 2, shields: 2}, false)).toEqual(ALL_OFF)
        expect(switchesHeld(enclose, {...NO_CHARGES, spreadClicksLeft: 2, shields: 2}, false)).toEqual(ALL_OFF)
        expect(switchesHeld(shield, {...NO_CHARGES, spreadClicksLeft: 2, enclosures: 2}, false)).toEqual(ALL_OFF)
    })

    it("keeps enclose on with no enclosure held while they are unlimited", () => {
        expect(switchesHeld(enclose, NO_CHARGES, true)).toBe(enclose)
        expect(switchesHeld(spread, NO_CHARGES, true)).toEqual(ALL_OFF)
    })

    it("never turns a switch on", () => {
        expect(switchesHeld(ALL_OFF, {...NO_CHARGES, enclosures: 3, spreadClicksLeft: 8, shields: 12}, false)).toBe(ALL_OFF)
    })
})

describe("atRest", () => {
    it("turns enclose on when nothing else is on and enclose rests on", () => {
        expect(atRest(ALL_OFF, true)).toEqual({...ALL_OFF, enclose: true})
    })

    it("leaves another switch alone", () => {
        const spread = {...ALL_OFF, spread: true}
        const shield = {...ALL_OFF, shield: true}

        expect(atRest(spread, true)).toBe(spread)
        expect(atRest(shield, true)).toBe(shield)
    })

    it("turns nothing on when enclose does not rest on", () => {
        expect(atRest(ALL_OFF, false)).toBe(ALL_OFF)
    })
})

describe("switched", () => {
    it("turns the others off: one bonus per click", () => {
        expect(switched(spread(), "enclose", true)).toEqual({...ALL_OFF, enclose: true})
        expect(switched({...ALL_OFF, enclose: true}, "spread", true)).toEqual(spread())
        expect(switched(spread(), "shield", true)).toEqual({...ALL_OFF, shield: true})
        expect(switched({...ALL_OFF, shield: true}, "enclose", true)).toEqual({...ALL_OFF, enclose: true})
    })

    it("turns one off and leaves the others alone", () => {
        expect(switched(spread(), "enclose", false)).toEqual(spread())
        expect(switched(spread(), "spread", false)).toEqual(ALL_OFF)
        expect(switched({...ALL_OFF, shield: true}, "shield", false)).toEqual(ALL_OFF)
    })
})

function spread() {
    return {...ALL_OFF, spread: true}
}
