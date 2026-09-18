import {describe, expect, it} from "vitest"
import {BonusReward, chargeLabels, describeReward, NO_CHARGES} from "./bonus.ts"

describe("describeReward", () => {
    it("says what a refill does", () => {
        const {title, detail} = describeReward({kind: "refill"})

        expect(title).toBe("Refill")
        expect(detail).toBe("Fills your clicks to full, when you choose")
    })

    it("says what a spread does and for how many clicks", () => {
        const {title, detail} = describeReward({kind: "spreadClicks", clicks: 8})

        expect(title).toBe("Spread clicks")
        expect(detail).toBe("Your next 8 clicks also take the tiles around them")
    })

    it("says what a bomb does and that it keeps", () => {
        const {title, detail} = describeReward({kind: "bomb", radius: 0.06})

        expect(title).toBe("Bomb")
        expect(detail).toBe("Resets the tiles in an area. Kept until you drop it")
    })

    it("says what an enclose does and how big a shape", () => {
        const {title, detail} = describeReward({kind: "encloseClicks", maxTiles: 25})

        expect(title).toBe("Enclose")
        expect(detail).toContain("25 tiles")
    })

    it("gives every kind a title of its own", () => {
        const rewards: BonusReward[] = [
            {kind: "refill"},
            {kind: "spreadClicks", clicks: 8},
            {kind: "bomb", radius: 0.06},
            {kind: "encloseClicks", maxTiles: 25},
        ]

        expect(new Set(rewards.map(reward => describeReward(reward).title)).size).toBe(rewards.length)
    })
})

describe("chargeLabels", () => {
    it("says nothing when nothing is held", () => {
        expect(chargeLabels(NO_CHARGES)).toEqual([])
    })

    it("says each charge held, the two to press first", () => {
        expect(chargeLabels({
            refill: true,
            bomb: true,
            enclose: true,
            spreadClicksLeft: 5,
        })).toEqual([
            {kind: "refill", label: "Refill ready"},
            {kind: "bomb", label: "Bomb ready"},
            {kind: "encloseClicks", label: "Enclose ready"},
            {kind: "spreadClicks", label: "Spread: 5 clicks left"},
        ])
    })

    it("counts the last spread click as one click", () => {
        expect(chargeLabels({...NO_CHARGES, spreadClicksLeft: 1})).toEqual([{kind: "spreadClicks", label: "Spread: 1 click left"}])
    })
})
