import {describe, expect, it} from "vitest"
import {ALL_OFF, atRest, CHARGE_KINDS, inUnlockOrder, lockedAt, NEVER, NO_CHARGES, NO_GATES, NO_LEVEL, restingSwitch, switched, switchesHeld} from "./bonus.ts"

const PAID = {spreadUnlimited: false, enclosuresUnlimited: false, enclosuresOff: false, spreadOff: false}

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

    it("turns spread off while spreads are off, held or unlimited", () => {
        const off = {...PAID, spreadOff: true}

        expect(switchesHeld(spread, {...NO_CHARGES, spreadClicksLeft: 5}, off)).toEqual(ALL_OFF)
        expect(switchesHeld(spread, NO_CHARGES, {...off, spreadUnlimited: true})).toEqual(ALL_OFF)
        expect(switchesHeld(enclose, {...NO_CHARGES, enclosures: 2}, off)).toBe(enclose)
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

    it("is never spread while spreads are off", () => {
        expect(restingSwitch({...PAID, spreadUnlimited: true, spreadOff: true})).toBeUndefined()
        expect(restingSwitch({...PAID, spreadUnlimited: true, spreadOff: true, enclosuresUnlimited: true})).toBe("enclose")
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
    const gates = {on: true, open: ["refill", "spreadClicks", "shields"] as const, unlocks: {encloseClicks: 4, bomb: 5}}

    it("names the level a gated kind opens at while the player is below it", () => {
        expect(lockedAt("bomb", gates, 3)).toBe(5)
        expect(lockedAt("encloseClicks", gates, NO_LEVEL)).toBe(4)
    })

    it("locks no kind at or past its level, nor an open kind", () => {
        expect(lockedAt("bomb", gates, 5)).toBeUndefined()
        expect(lockedAt("encloseClicks", gates, 9)).toBeUndefined()
        expect(lockedAt("refill", gates, NO_LEVEL)).toBeUndefined()
    })

    it("locks a gated kind the ladder names no level for at every level", () => {
        const omitted = {...gates, unlocks: {encloseClicks: 4}}

        expect(lockedAt("bomb", omitted, 99)).toBe(NEVER)
    })

    it("locks nothing while the gates are off", () => {
        expect(lockedAt("bomb", {...gates, on: false}, NO_LEVEL)).toBeUndefined()
        expect(lockedAt("bomb", NO_GATES, NO_LEVEL)).toBeUndefined()
    })
})

describe("inUnlockOrder", () => {
    it("puts the open kinds first, then the gated ones by the level they open at, the order given among one level", () => {
        const gates = {on: true, open: ["refill", "spreadClicks", "shields"] as const, unlocks: {encloseClicks: 4, bomb: 5}}

        expect(inUnlockOrder(CHARGE_KINDS, gates)).toEqual(["refill", "spreadClicks", "shields", "encloseClicks", "bomb", "virus", "mine", "rally"])
    })

    it("puts last a gated kind the ladder names no level for", () => {
        const gates = {on: true, open: ["refill", "spreadClicks", "shields"] as const, unlocks: {bomb: 5}}

        expect(inUnlockOrder(CHARGE_KINDS, gates)).toEqual(["refill", "spreadClicks", "shields", "bomb", "encloseClicks", "virus", "mine", "rally"])
    })

    it("keeps the order given while no open kind is told, whatever the ladder", () => {
        expect(inUnlockOrder(CHARGE_KINDS, {on: false, open: [], unlocks: {encloseClicks: 4, bomb: 5}})).toEqual(CHARGE_KINDS)
    })

    it("orders by the ladder with the gates off too", () => {
        const gates = {on: false, open: ["refill", "spreadClicks", "shields"] as const, unlocks: {encloseClicks: 4, bomb: 5}}

        expect(inUnlockOrder(CHARGE_KINDS, gates)).toEqual(["refill", "spreadClicks", "shields", "encloseClicks", "bomb", "virus", "mine", "rally"])
    })

    it("keeps the order given with no gates", () => {
        expect(inUnlockOrder(CHARGE_KINDS, NO_GATES)).toEqual(CHARGE_KINDS)
    })
})
