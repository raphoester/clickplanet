import {describe, expect, it} from "vitest"
import {ALL_OFF, atRest, CHARGE_KINDS, FIRST_LEVEL, inUnlockOrder, lockedAt, NO_CHARGES, restingSwitch, switched, switchesHeld} from "./bonus.ts"

const PAID = {spreadUnlimited: false, enclosuresUnlimited: false, enclosuresOff: false}

describe("switchesHeld", () => {
    const spread = {...ALL_OFF, spread: true}
    const enclose = {...ALL_OFF, enclose: true}
    const shield = {...ALL_OFF, shield: true}

    it("keeps a switch on while its pool holds something", () => {
        const held = {...NO_CHARGES, enclosures: 1, spreadClicksLeft: 4, shields: 2}

        expect(switchesHeld(spread, held, PAID)).toBe(spread)
        expect(switchesHeld(enclose, held, PAID)).toBe(enclose)
        expect(switchesHeld(shield, held, PAID)).toBe(shield)
    })

    it("turns a switch off once its pool is empty", () => {
        expect(switchesHeld(spread, {...NO_CHARGES, enclosures: 2, shields: 2}, PAID)).toEqual(ALL_OFF)
        expect(switchesHeld(enclose, {...NO_CHARGES, spreadClicksLeft: 2, shields: 2}, PAID)).toEqual(ALL_OFF)
        expect(switchesHeld(shield, {...NO_CHARGES, spreadClicksLeft: 2, enclosures: 2}, PAID)).toEqual(ALL_OFF)
    })

    it("keeps enclose on with no enclosure held while they are unlimited", () => {
        const unlimited = {...PAID, enclosuresUnlimited: true}

        expect(switchesHeld(enclose, NO_CHARGES, unlimited)).toBe(enclose)
        expect(switchesHeld(spread, NO_CHARGES, unlimited)).toEqual(ALL_OFF)
    })

    it("keeps spread on with no spread click held while they are unlimited", () => {
        const unlimited = {...PAID, spreadUnlimited: true}

        expect(switchesHeld(spread, NO_CHARGES, unlimited)).toBe(spread)
        expect(switchesHeld(enclose, NO_CHARGES, unlimited)).toEqual(ALL_OFF)
    })

    it("turns enclose off while enclosures are off, held or not", () => {
        const off = {...PAID, enclosuresOff: true}

        expect(switchesHeld(enclose, {...NO_CHARGES, enclosures: 3}, off)).toEqual(ALL_OFF)
        expect(switchesHeld(spread, {...NO_CHARGES, spreadClicksLeft: 2}, off)).toBe(spread)
    })

    it("never turns a switch on", () => {
        expect(switchesHeld(ALL_OFF, {...NO_CHARGES, enclosures: 3, spreadClicksLeft: 8, shields: 12}, PAID)).toBe(ALL_OFF)
    })
})

describe("atRest", () => {
    it("turns the resting switch on when nothing else is on", () => {
        expect(atRest(ALL_OFF, "enclose")).toEqual({...ALL_OFF, enclose: true})
        expect(atRest(ALL_OFF, "spread")).toEqual({...ALL_OFF, spread: true})
    })

    it("leaves another switch alone", () => {
        const spread = {...ALL_OFF, spread: true}
        const shield = {...ALL_OFF, shield: true}

        expect(atRest(spread, "enclose")).toBe(spread)
        expect(atRest(shield, "spread")).toBe(shield)
    })

    it("turns nothing on when no switch rests", () => {
        expect(atRest(ALL_OFF, undefined)).toBe(ALL_OFF)
    })
})

describe("restingSwitch", () => {
    it("is the unlimited one, spread first", () => {
        expect(restingSwitch(PAID)).toBeUndefined()
        expect(restingSwitch(undefined)).toBeUndefined()
        expect(restingSwitch({...PAID, enclosuresUnlimited: true})).toBe("enclose")
        expect(restingSwitch({...PAID, spreadUnlimited: true})).toBe("spread")
        expect(restingSwitch({...PAID, spreadUnlimited: true, enclosuresUnlimited: true})).toBe("spread")
    })

    it("is never enclose while enclosures are off", () => {
        expect(restingSwitch({...PAID, enclosuresUnlimited: true, enclosuresOff: true})).toBeUndefined()
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

describe("lockedAt", () => {
    const unlocks = {encloseClicks: 4, bomb: 5}

    it("names the level a kind opens at while the player is below it", () => {
        expect(lockedAt("bomb", unlocks, 3)).toBe(5)
        expect(lockedAt("encloseClicks", unlocks, 3)).toBe(4)
    })

    it("locks nothing at or past its level, nor a kind the rules name no level for", () => {
        expect(lockedAt("bomb", unlocks, 5)).toBeUndefined()
        expect(lockedAt("encloseClicks", unlocks, 9)).toBeUndefined()
        expect(lockedAt("refill", unlocks, FIRST_LEVEL)).toBeUndefined()
    })
})

describe("inUnlockOrder", () => {
    it("puts the kinds by the level they open at, the order given among one level", () => {
        expect(inUnlockOrder(CHARGE_KINDS, {encloseClicks: 4, bomb: 5}))
            .toEqual(["refill", "spreadClicks", "shields", "encloseClicks", "bomb"])
    })

    it("keeps the order given while every kind opens at the first level", () => {
        expect(inUnlockOrder(CHARGE_KINDS, {})).toEqual(CHARGE_KINDS)
    })
})
