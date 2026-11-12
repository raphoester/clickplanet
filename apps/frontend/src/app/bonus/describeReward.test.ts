import {describe, expect, it} from "vitest"
import {BonusReward} from "../../domain/bonus.ts"
import {describeReward} from "./describeReward.ts"

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
        const {title, detail} = describeReward({kind: "encloseClicks", shapes: 2, maxTiles: 1500})

        expect(title).toBe("+2 enclosures")
        expect(detail).toBe("Switch enclose on, then close a shape of up to 1,500 tiles to take the tiles inside")
        expect(describeReward({kind: "encloseClicks", shapes: 1, maxTiles: 25}).title).toBe("+1 enclosure")
    })

    it("says how many shields a box added, and where they go", () => {
        const {title, detail} = describeReward({kind: "shields", shields: 3})

        expect(title).toBe("+3 shields")
        expect(detail).toBe("Switch shield on, then tap your tiles. Each shield stops one enemy click")
        expect(describeReward({kind: "shields", shields: 1}).title).toBe("+1 shield")
    })

    it("gives every kind a title of its own", () => {
        const rewards: BonusReward[] = [
            {kind: "refill"},
            {kind: "spreadClicks", clicks: 1},
            {kind: "bomb", radius: 0.06},
            {kind: "encloseClicks", shapes: 1, maxTiles: 25},
            {kind: "shields", shields: 1},
        ]

        expect(new Set(rewards.map(reward => describeReward(reward).title)).size).toBe(rewards.length)
    })
})
