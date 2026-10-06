import {describe, expect, it} from "vitest"
import {ALL_OFF, BonusReward, describeReward, NO_CHARGES, switched, switchesHeld} from "./bonus.ts"

describe("describeReward", () => {
    it("says what a refill does", () => {
        const {title, detail} = describeReward({kind: "refill"})

        expect(title).toBe("Refill")
        expect(detail).toBe("Fills your clicks to full, when you choose")
    })

    it("says how many spread clicks a box added, and that spread is switched on", () => {
        const {title, detail} = describeReward({kind: "spreadClicks", clicks: 3})

        expect(title).toBe("+3 spread clicks")
        expect(detail).toBe("Switch spread on: each click also takes the tiles around it")
        expect(describeReward({kind: "spreadClicks", clicks: 1}).title).toBe("+1 spread click")
    })

    it("says what a bomb does and where it is aimed from", () => {
        const {title, detail} = describeReward({kind: "bomb", radius: 0.06})

        expect(title).toBe("Bomb")
        expect(detail).toBe("Resets the tiles in an area. Aim it from your inventory")
    })

    it("says how many enclosures a box added, and how big a shape", () => {
        const {title, detail} = describeReward({kind: "encloseClicks", shapes: 2, maxTiles: 25})

        expect(title).toBe("+2 enclosures")
        expect(detail).toContain("25 tiles")
        expect(describeReward({kind: "encloseClicks", shapes: 1, maxTiles: 25}).title).toBe("+1 enclosure")
    })

    it("says how many defenders a box added, and nothing more", () => {
        const {title, detail} = describeReward({kind: "defenders", defenders: 3})

        expect(title).toBe("+3 defenders")
        expect(detail).toBeUndefined()
        expect(describeReward({kind: "defenders", defenders: 1}).title).toBe("+1 defender")
    })

    it("gives every kind a title of its own", () => {
        const rewards: BonusReward[] = [
            {kind: "refill"},
            {kind: "spreadClicks", clicks: 1},
            {kind: "bomb", radius: 0.06},
            {kind: "encloseClicks", shapes: 1, maxTiles: 25},
            {kind: "defenders", defenders: 1},
        ]

        expect(new Set(rewards.map(reward => describeReward(reward).title)).size).toBe(rewards.length)
    })
})

describe("switchesHeld", () => {
    const spread = {...ALL_OFF, spread: true}
    const enclose = {...ALL_OFF, enclose: true}
    const defend = {...ALL_OFF, defend: true}

    it("keeps a switch on while its pool holds something", () => {
        const held = {...NO_CHARGES, enclosures: 1, spreadClicksLeft: 4, defenders: 2}

        expect(switchesHeld(spread, held)).toBe(spread)
        expect(switchesHeld(enclose, held)).toBe(enclose)
        expect(switchesHeld(defend, held)).toBe(defend)
    })

    it("turns a switch off once its pool is empty", () => {
        expect(switchesHeld(spread, {...NO_CHARGES, enclosures: 2, defenders: 2})).toEqual(ALL_OFF)
        expect(switchesHeld(enclose, {...NO_CHARGES, spreadClicksLeft: 2, defenders: 2})).toEqual(ALL_OFF)
        expect(switchesHeld(defend, {...NO_CHARGES, spreadClicksLeft: 2, enclosures: 2})).toEqual(ALL_OFF)
    })

    it("never turns a switch on", () => {
        expect(switchesHeld(ALL_OFF, {...NO_CHARGES, enclosures: 3, spreadClicksLeft: 8, defenders: 12})).toBe(ALL_OFF)
    })
})

describe("switched", () => {
    it("turns the others off: one bonus per click", () => {
        expect(switched(spread(), "enclose", true)).toEqual({...ALL_OFF, enclose: true})
        expect(switched({...ALL_OFF, enclose: true}, "spread", true)).toEqual(spread())
        expect(switched(spread(), "defend", true)).toEqual({...ALL_OFF, defend: true})
        expect(switched({...ALL_OFF, defend: true}, "enclose", true)).toEqual({...ALL_OFF, enclose: true})
    })

    it("turns one off and leaves the others alone", () => {
        expect(switched(spread(), "enclose", false)).toEqual(spread())
        expect(switched(spread(), "spread", false)).toEqual(ALL_OFF)
        expect(switched({...ALL_OFF, defend: true}, "defend", false)).toEqual(ALL_OFF)
    })
})

function spread() {
    return {...ALL_OFF, spread: true}
}
